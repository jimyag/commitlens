package cmd

import (
	"context"
	"testing"

	"github.com/jimyag/commitlens/internal/config"
)

func TestResolveRepositoriesIgnoresOrganizationsOutsideWebMode(t *testing.T) {
	cfg := &config.Config{
		Repositories: []config.Repository{
			{Owner: "example-org"},
			{Owner: "example-org", Repo: "alpha"},
			{Owner: "example-org", Repo: "alpha"},
		},
	}

	repositories, err := resolveRepositories(context.Background(), cfg, false)
	if err != nil {
		t.Fatalf("resolveRepositories() error: %v", err)
	}
	if len(repositories.available) != 1 || repositories.available[0].ID() != "example-org/alpha" {
		t.Fatalf("unexpected available repositories: %#v", repositories.available)
	}
	if len(repositories.startup) != 1 || repositories.startup[0].ID() != "example-org/alpha" {
		t.Fatalf("unexpected startup repositories: %#v", repositories.startup)
	}
}
