package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListOrganizationRepositories(t *testing.T) {
	var sawAuthorization bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs/example-org/repos" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") == "Bearer test-token" {
			sawAuthorization = true
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"name":"alpha"},{"name":"beta"}]`)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.httpClient = server.Client()
	client.baseURL = server.URL

	repositories, err := client.ListOrganizationRepositories(context.Background(), "example-org")
	if err != nil {
		t.Fatalf("ListOrganizationRepositories() error: %v", err)
	}
	if !sawAuthorization {
		t.Fatal("expected Authorization header")
	}
	if len(repositories) != 2 ||
		repositories[0].ID() != "example-org/alpha" ||
		repositories[1].ID() != "example-org/beta" {
		t.Fatalf("unexpected repositories: %#v", repositories)
	}
}
