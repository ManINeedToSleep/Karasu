package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const fanartBaseURL = "https://webservice.fanart.tv/v3/music"

// FanartClient handles communication with the Fanart.tv API
type FanartClient struct {
	apiKey string
	http   *http.Client
}

// NewFanartClient creates a new Fanart.tv client
func NewFanartClient(apiKey string) *FanartClient {
	return &FanartClient{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

// -----------------------------------------------------------------------------
// Artist images
// -----------------------------------------------------------------------------

type fanartImage struct {
	URL   string `json:"url"`
	Likes string `json:"likes"`
}

type fanartArtistResponse struct {
	ArtistThumb      []fanartImage `json:"artistthumb"`
	ArtistBackground []fanartImage `json:"artistbackground"`
	HDMusicLogo      []fanartImage `json:"hdmusiclogo"`
	MusicBanner      []fanartImage `json:"musicbanner"`
}

// GetArtistImageURL returns the best artist image URL from Fanart.tv
// Uses the MusicBrainz ID for lookup
func (c *FanartClient) GetArtistImageURL(mbid string) (string, error) {
	url := fmt.Sprintf("%s/%s?api_key=%s", fanartBaseURL, mbid, c.apiKey)

	resp, err := c.http.Get(url)
	if err != nil {
		return "", fmt.Errorf("fanart request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", fmt.Errorf("no fanart found for mbid %s", mbid)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fanart returned status %d", resp.StatusCode)
	}

	var result fanartArtistResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode fanart response: %w", err)
	}

	// Prefer artistthumb (square, perfect for our grid)
	// Fall back to artistbackground
	if len(result.ArtistThumb) > 0 {
		return result.ArtistThumb[0].URL, nil
	}

	if len(result.ArtistBackground) > 0 {
		return result.ArtistBackground[0].URL, nil
	}

	return "", fmt.Errorf("no suitable image found for mbid %s", mbid)
}