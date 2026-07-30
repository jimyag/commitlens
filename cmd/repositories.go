package cmd

import (
	"context"
	"fmt"

	"github.com/jimyag/commitlens/internal/config"
	"github.com/jimyag/commitlens/internal/git"
	ghapi "github.com/jimyag/commitlens/internal/github"
)

type repositorySet struct {
	available []config.Repository
	startup   []config.Repository
}

func resolveRepositories(ctx context.Context, cfg *config.Config, includeOrganizations bool) (repositorySet, error) {
	var startup []config.Repository
	var organizations []string
	for _, entry := range cfg.Repositories {
		switch {
		case entry.IsOrganization():
			organizations = append(organizations, entry.Owner)
		case entry.IsSyncable():
			startup = append(startup, entry)
		}
	}

	if len(cfg.DiscoveryRoots) > 0 {
		discovered, err := git.Discover(cfg.DiscoveryRoots)
		if err != nil {
			return repositorySet{}, fmt.Errorf("discovery failed: %w", err)
		}
		startup = append(startup, discovered...)
	}
	startup = deduplicateRepositories(startup)
	available := append([]config.Repository(nil), startup...)

	if includeOrganizations && len(organizations) > 0 {
		client := ghapi.NewClient(cfg.GitHub.Token)
		for _, owner := range uniqueStrings(organizations) {
			repositories, err := client.ListOrganizationRepositories(ctx, owner)
			if err != nil {
				return repositorySet{}, err
			}
			available = append(available, repositories...)
		}
	}

	return repositorySet{
		available: deduplicateRepositories(available),
		startup:   startup,
	}, nil
}

func deduplicateRepositories(repositories []config.Repository) []config.Repository {
	unique := make([]config.Repository, 0, len(repositories))
	seen := make(map[string]struct{}, len(repositories))
	for _, repository := range repositories {
		if !repository.IsSyncable() {
			continue
		}
		id := repository.ID()
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, repository)
	}
	return unique
}

func uniqueStrings(values []string) []string {
	unique := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}
