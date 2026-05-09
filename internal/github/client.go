package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type PR struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	AvatarURL string    `json:"avatar_url"`
	MergedAt  time.Time `json:"merged_at"`
	Additions int       `json:"additions"`
	Deletions int       `json:"deletions"`
	Commits   []Commit  `json:"commits"`
}

type Commit struct {
	SHA       string    `json:"sha"`
	Author    string    `json:"author"`
	Message   string    `json:"message"`
	Additions int       `json:"additions"`
	Deletions int       `json:"deletions"`
	Date      time.Time `json:"date"`
}

// FetchProgress is reported during GetMergedPRsSince.
type FetchProgress struct {
	PRsFetched  int
	PRsTotal    int // -1 during list phase; >=0 in detail phase
	CommitsDone int
	// ListPage is the current GitHub list API page (1-based) when PRsTotal < 0; 0 in detail phase.
	ListPage int
	// Log is a short English status line (e.g. list paging); optional; shown by sync UI.
	Log string
}

type Client struct {
	token       string
	baseURL     string
	httpClient  *http.Client
	concurrency int
}

func NewClient(token string) *Client {
	if token == "" {
		token = tokenFromGH()
	}
	return &Client{
		token:       token,
		baseURL:     "https://api.github.com",
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		concurrency: 16,
	}
}

// SetConcurrency controls how many commit-fetch requests run in parallel.
func (c *Client) SetConcurrency(n int) {
	if n > 0 {
		c.concurrency = n
	}
}

func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

func tokenFromGH() string {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("github API error: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// GetMergedPRsSince fetches all merged PRs since the given time, then fetches
// their commits concurrently. onProgress is called after each batch completes;
// pass nil to skip progress reporting.
func (c *Client) GetMergedPRsSince(ctx context.Context, owner, repo string, since time.Time, onProgress func(FetchProgress)) ([]PR, error) {
	// Phase 1: collect PR stubs page by page
	var stubs []PR
	page := 1
	for {
		var raw []struct {
			Number    int    `json:"number"`
			Title     string `json:"title"`
			User      struct {
				Login     string `json:"login"`
				AvatarURL string `json:"avatar_url"`
			} `json:"user"`
			MergedAt  *time.Time `json:"merged_at"`
			UpdatedAt time.Time  `json:"updated_at"`
		}
		path := fmt.Sprintf("/repos/%s/%s/pulls?state=closed&per_page=100&page=%d&sort=updated&direction=desc", owner, repo, page)
		if err := c.get(ctx, path, &raw); err != nil {
			return nil, err
		}
		if len(raw) == 0 {
			break
		}
		done := false
		for _, r := range raw {
			// Stop paging when updated_at falls before our cutoff.
			// Use updated_at (not merged_at) because sort order is by updated.
			if !since.IsZero() && r.UpdatedAt.Before(since) {
				done = true
				break
			}
			// Only include merged PRs (merged_at is null for closed-unmerged).
			if r.MergedAt == nil {
				continue
			}
			stubs = append(stubs, PR{
				Number:    r.Number,
				Title:     r.Title,
				Author:    r.User.Login,
				AvatarURL: r.User.AvatarURL,
				MergedAt:  *r.MergedAt,
			})
		}
		if done {
			break
		}
		if onProgress != nil && len(stubs) > 0 {
			onProgress(FetchProgress{
				PRsFetched: len(stubs),
				PRsTotal:   -1,
				ListPage:   page,
				Log:        fmt.Sprintf("list page %d, %d PRs", page, len(stubs)),
			})
		}
		page++
	}

	if len(stubs) == 0 {
		return nil, nil
	}

	if onProgress != nil {
		onProgress(FetchProgress{
			PRsFetched: 0,
			PRsTotal:   len(stubs),
			ListPage:   0,
			Log:        "fetch PR details & commits",
		})
	}

	// Phase 2: fetch PR details (additions/deletions) and commits concurrently.
	// The list API omits additions/deletions; the individual PR endpoint has them.
	results := make([]PR, len(stubs))
	var (
		mu   sync.Mutex
		done int
	)

	sem := make(chan struct{}, c.concurrency)
	g, gctx := errgroup.WithContext(ctx)

	for i, stub := range stubs {
		i, stub := i, stub
		sem <- struct{}{}
		g.Go(func() error {
			defer func() { <-sem }()

			// Fetch additions/deletions from individual PR endpoint
			if add, del, err := c.getPRStats(gctx, owner, repo, stub.Number); err == nil {
				stub.Additions = add
				stub.Deletions = del
			}

			// Fetch commits (non-fatal if it fails)
			commits, _ := c.GetPRCommits(gctx, owner, repo, stub.Number)
			stub.Commits = commits
			results[i] = stub

			if onProgress != nil {
				mu.Lock()
				done++
				n := len(stubs)
				// Throttle: avoid thousands of channel sends and TUI frame backlog on huge repos; always send 1, last, and periodic samples.
				step := max(1, n/200)
				if n <= 20 || done == 1 || done == n || done%step == 0 {
					onProgress(FetchProgress{
						PRsFetched:  done,
						PRsTotal:    n,
						CommitsDone: done,
						ListPage:    0,
						Log:         "",
					})
				}
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// getPRStats fetches additions and deletions for a single PR.
func (c *Client) getPRStats(ctx context.Context, owner, repo string, prNumber int) (additions, deletions int, err error) {
	var raw struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, prNumber)
	if err = c.get(ctx, path, &raw); err != nil {
		return 0, 0, err
	}
	return raw.Additions, raw.Deletions, nil
}

func (c *Client) GetPRCommits(ctx context.Context, owner, repo string, prNumber int) ([]Commit, error) {
	var raw []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name string    `json:"name"`
				Date time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
		Author *struct {
			Login string `json:"login"`
		} `json:"author"`
		Stats struct {
			Additions int `json:"additions"`
			Deletions int `json:"deletions"`
		} `json:"stats"`
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/commits?per_page=100", owner, repo, prNumber)
	if err := c.get(ctx, path, &raw); err != nil {
		return nil, err
	}
	commits := make([]Commit, 0, len(raw))
	for _, r := range raw {
		author := r.Commit.Author.Name
		if r.Author != nil {
			author = r.Author.Login
		}
		commits = append(commits, Commit{
			SHA:       r.SHA,
			Author:    author,
			Message:   r.Commit.Message,
			Additions: r.Stats.Additions,
			Deletions: r.Stats.Deletions,
			Date:      r.Commit.Author.Date,
		})
	}
	return commits, nil
}

func (c *Client) graphql(ctx context.Context, query string, variables map[string]interface{}, out any) error {
	body := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2s, 4s, 8s
			time.Sleep(time.Duration(1<<attempt) * time.Second)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/graphql", strings.NewReader(string(data)))
		if err != nil {
			return err
		}
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("github GraphQL error: %s", resp.Status)
			resp.Body.Close()
			continue
		}

		if resp.StatusCode >= 400 {
			defer resp.Body.Close()
			return fmt.Errorf("github GraphQL error: %s", resp.Status)
		}

		var result struct {
			Data   json.RawMessage `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()

		if len(result.Errors) > 0 {
			// Some GraphQL errors might be transient, but we'll return immediately for now
			return fmt.Errorf("graphql error: %s", result.Errors[0].Message)
		}

		return json.Unmarshal(result.Data, out)
	}

	return lastErr
}

func (c *Client) GetDirectCommitsSince(ctx context.Context, owner, repo string, since time.Time, onProgress func(FetchProgress)) ([]Commit, error) {
	query := `
query($owner: String!, $name: String!, $since: GitTimestamp, $cursor: String) {
  repository(owner: $owner, name: $name) {
    defaultBranchRef {
      target {
        ... on Commit {
          history(since: $since, first: 100, after: $cursor) {
            pageInfo { hasNextPage endCursor }
            nodes {
              oid
              messageHeadline
              committedDate
              additions
              deletions
              author {
                user { login }
                name
              }
              associatedPullRequests(first: 1) {
                totalCount
              }
            }
          }
        }
      }
    }
  }
}
`
	var commits []Commit
	var cursor *string
	page := 1

	for {
		variables := map[string]interface{}{
			"owner": owner,
			"name":  repo,
		}
		if cursor != nil {
			variables["cursor"] = *cursor
		}
		if !since.IsZero() {
			variables["since"] = since.Format(time.RFC3339)
		}

		var resp struct {
			Repository struct {
				DefaultBranchRef *struct {
					Target struct {
						History struct {
							PageInfo struct {
								HasNextPage bool    `json:"hasNextPage"`
								EndCursor   *string `json:"endCursor"`
							} `json:"pageInfo"`
							Nodes []struct {
								Oid             string    `json:"oid"`
								MessageHeadline string    `json:"messageHeadline"`
								CommittedDate   time.Time `json:"committedDate"`
								Additions       int       `json:"additions"`
								Deletions       int       `json:"deletions"`
								Author          struct {
									User *struct {
										Login string `json:"login"`
									} `json:"user"`
									Name string `json:"name"`
								} `json:"author"`
								AssociatedPullRequests struct {
									TotalCount int `json:"totalCount"`
								} `json:"associatedPullRequests"`
							} `json:"nodes"`
						} `json:"history"`
					} `json:"target"`
				} `json:"defaultBranchRef"`
			} `json:"repository"`
		}

		if err := c.graphql(ctx, query, variables, &resp); err != nil {
			return nil, err
		}

		ref := resp.Repository.DefaultBranchRef
		if ref == nil {
			break // Empty repository or no default branch
		}

		nodes := ref.Target.History.Nodes
		for _, node := range nodes {
			if node.AssociatedPullRequests.TotalCount == 0 {
				author := node.Author.Name
				if node.Author.User != nil {
					author = node.Author.User.Login
				}
				commits = append(commits, Commit{
					SHA:       node.Oid,
					Author:    author,
					Message:   node.MessageHeadline,
					Additions: node.Additions,
					Deletions: node.Deletions,
					Date:      node.CommittedDate,
				})
			}
		}

		if onProgress != nil && len(nodes) > 0 {
			onProgress(FetchProgress{
				PRsFetched: len(commits),
				PRsTotal:   -1,
				ListPage:   page,
				Log:        fmt.Sprintf("checking commits page %d", page),
			})
		}

		if !ref.Target.History.PageInfo.HasNextPage {
			break
		}
		cursor = ref.Target.History.PageInfo.EndCursor
		page++
	}

	return commits, nil
}
