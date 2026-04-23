package github_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jimyag/commitlens/internal/github"
)

func TestClient_GetMergedPRsSince(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: pulls list
			prs := []map[string]any{
				{
					"number":    1,
					"title":     "test pr",
					"user":      map[string]any{"login": "jimyag", "avatar_url": "https://example.com/avatar"},
					"merged_at": time.Now().Format(time.RFC3339),
					"additions": 100,
					"deletions": 20,
				},
			}
			json.NewEncoder(w).Encode(prs)
		} else {
			// Subsequent calls: commits list (empty)
			json.NewEncoder(w).Encode([]any{})
		}
	}))
	defer srv.Close()

	client := github.NewClient("fake-token")
	client.SetBaseURL(srv.URL)

	since := time.Now().Add(-24 * time.Hour)
	prs, err := client.GetMergedPRsSince(context.Background(), "jimyag", "commitlens", since, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) == 0 {
		t.Fatal("expected at least 1 PR")
	}
	if prs[0].Author != "jimyag" {
		t.Errorf("expected author jimyag, got %s", prs[0].Author)
	}
}
