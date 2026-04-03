package slskd

import (
	"fmt"
	"testing"
	"time"
)

func TestSearch(t *testing.T) {
	client := NewClient(
		"http://localhost:5030",
		"neosgw",
		"waljyaCMv7NLlG",
	)

	search, err := client.StartSearch("Taylor Swift 1989")
	if err != nil {
		t.Fatalf("Failed to start search: %v", err)
	}
	fmt.Printf("Search started! ID: %s\n", search.ID)

	if err := client.WaitForSearch(search.ID, 60*time.Second); err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	results, err := client.GetSearchResults(search.ID)
	if err != nil {
		t.Fatalf("Failed to get results: %v", err)
	}

	fmt.Printf("Found %d users with results\n\n", len(results))
	for i, r := range results {
		if i >= 3 { break }
		fmt.Printf("User: %-30s | Files: %d\n", r.Username, len(r.Files))
	}

	if len(results) == 0 {
		t.Error("Expected results but got none")
	}
}