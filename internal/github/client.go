package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jimyag/commitlens/internal/config"
)

const defaultAPIURL = "https://api.github.com"

type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultAPIURL,
		token:      strings.TrimSpace(token),
	}
}

func ResolveToken(configured string) string {
	if token := strings.TrimSpace(configured); token != "" {
		return token
	}
	for _, key := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(key)); token != "" {
			return token
		}
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (c *Client) ListOrganizationRepositories(ctx context.Context, owner string) ([]config.Repository, error) {
	const perPage = 100

	var repositories []config.Repository
	for page := 1; ; page++ {
		endpoint := fmt.Sprintf(
			"%s/orgs/%s/repos?type=all&sort=full_name&per_page=%d&page=%d",
			strings.TrimRight(c.baseURL, "/"),
			url.PathEscape(owner),
			perPage,
			page,
		)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list repositories for %s: %w", owner, err)
		}

		var pageRepos []struct {
			Name string `json:"name"`
		}
		if resp.StatusCode == http.StatusOK {
			err = json.NewDecoder(resp.Body).Decode(&pageRepos)
		} else {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			err = fmt.Errorf(
				"list repositories for %s: GitHub API returned %s: %s",
				owner,
				resp.Status,
				strings.TrimSpace(string(body)),
			)
		}
		if closeErr := resp.Body.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return nil, err
		}

		for _, repo := range pageRepos {
			if repo.Name != "" {
				repositories = append(repositories, config.Repository{Owner: owner, Repo: repo.Name})
			}
		}
		if len(pageRepos) < perPage {
			break
		}
	}
	return repositories, nil
}
