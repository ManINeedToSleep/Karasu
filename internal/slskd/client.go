package slskd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client handles all communication with the slskd REST API
type Client struct {
	baseURL  string
	username string
	password string
	token    string
	tokenExp time.Time
	http     *http.Client
}

// NewClient creates a new slskd API client
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

// -----------------------------------------------------------------------------
// Authentication
// -----------------------------------------------------------------------------

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
}

// login authenticates with slskd and stores the JWT token
func (c *Client) login() error {
	body, _ := json.Marshal(loginRequest{Username: c.username, Password: c.password})

	resp, err := c.http.Post(
		fmt.Sprintf("%s/api/v0/session", c.baseURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var result loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	c.token = result.Token
	// Store expiry time so we know when to re-login
	c.tokenExp = time.Unix(result.Expires, 0)

	return nil
}

// ensureAuth makes sure we have a valid token, re-logging in if needed
func (c *Client) ensureAuth() error {
	// Re-login if we have no token or it expires within the next 5 minutes
	if c.token == "" || time.Until(c.tokenExp) < 5*time.Minute {
		return c.login()
	}
	return nil
}

// -----------------------------------------------------------------------------
// HTTP helpers
// -----------------------------------------------------------------------------

// do makes an authenticated HTTP request
func (c *Client) do(method, endpoint string, body any, out any) error {
	if err := c.ensureAuth(); err != nil {
		return fmt.Errorf("auth failed: %w", err)
	}

	var bodyReader *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(b)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.baseURL, endpoint), bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slskd returned status %d for %s %s", resp.StatusCode, method, endpoint)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// Searches
// -----------------------------------------------------------------------------

// SearchRequest is the body sent to start a search
type SearchRequest struct {
	SearchText string `json:"searchText"`
}

// SearchResponse is what slskd returns when a search is created
type SearchResponse struct {
	ID         string `json:"id"`
	State      string `json:"state"`
	IsComplete bool   `json:"isComplete"`
	FileCount  int    `json:"fileCount"`
}

// FileResult represents a single file found in a search result
type FileResult struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	BitRate     int    `json:"bitRate"`
	SampleRate  int    `json:"sampleRate"`
	BitDepth    int    `json:"bitDepth"`
	IsVariableBitRate bool `json:"isVariableBitRate"`
}

// SearchResult represents one user's response to a search
type SearchResult struct {
	Username  string       `json:"username"`
	Files     []FileResult `json:"files"`
	FreeUploadSlots int    `json:"freeUploadSlots"`
	UploadSpeed     int    `json:"uploadSpeed"`
}

// StartSearch begins a search on Soulseek and returns the search ID
func (c *Client) StartSearch(query string) (*SearchResponse, error) {
	var result SearchResponse
	err := c.do("POST", "/api/v0/searches", SearchRequest{SearchText: query}, &result)
	if err != nil {
		return nil, fmt.Errorf("start search failed: %w", err)
	}
	return &result, nil
}

// GetSearchState checks whether a search is still running or complete
func (c *Client) GetSearchState(searchID string) (*SearchResponse, error) {
	var result SearchResponse
	err := c.do("GET", fmt.Sprintf("/api/v0/searches/%s", searchID), nil, &result)
	if err != nil {
		return nil, fmt.Errorf("get search state failed: %w", err)
	}
	return &result, nil
}

// GetSearchResults returns all file results for a completed search
func (c *Client) GetSearchResults(searchID string) ([]SearchResult, error) {
	var results []SearchResult
	err := c.do("GET", fmt.Sprintf("/api/v0/searches/%s/responses", searchID), nil, &results)
	if err != nil {
		return nil, fmt.Errorf("get search results failed: %w", err)
	}
	return results, nil
}

// WaitForSearch polls until a search is complete or times out
func (c *Client) WaitForSearch(searchID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		state, err := c.GetSearchState(searchID)
		if err != nil {
			return err
		}
		// State can be "Completed", "TimedOut", or "Completed, TimedOut"
		if state.State == "Completed" || state.State == "TimedOut" ||
			state.State == "Completed, TimedOut" || state.IsComplete {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("search timed out after %s", timeout)
}

// -----------------------------------------------------------------------------
// Downloads
// -----------------------------------------------------------------------------

// DownloadRequest is the body sent to enqueue a download
type DownloadRequest struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// EnqueueDownload tells slskd to download a specific file from a specific user
// slskd expects an array of requests
func (c *Client) EnqueueDownload(username, filename string, size int64) error {
	return c.do(
		"POST",
		fmt.Sprintf("/api/v0/transfers/downloads/%s", username),
		[]DownloadRequest{{Filename: filename, Size: size}},
		nil,
	)
}

// Transfer represents a download/upload in progress or completed
type Transfer struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	State    string `json:"state"`
	Size     int64  `json:"size"`
	BytesTransferred int64 `json:"bytesTransferred"`
}

// GetAllDownloads returns all current and recent downloads
func (c *Client) GetAllDownloads() ([]Transfer, error) {
	var transfers []Transfer
	err := c.do("GET", "/api/v0/transfers/downloads", nil, &transfers)
	if err != nil {
		return nil, fmt.Errorf("get downloads failed: %w", err)
	}
	return transfers, nil
}