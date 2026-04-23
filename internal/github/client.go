package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
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
	SHA       string `json:"sha"`
	Author    string `json:"author"`
	Message   string `json:"message"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	if token == "" {
		token = tokenFromGH()
	}
	return &Client{
		token:      token,
		baseURL:    "https://api.github.com",
		httpClient: &http.Client{Timeout: 30 * time.Second},
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

func (c *Client) GetMergedPRsSince(ctx context.Context, owner, repo string, since time.Time) ([]PR, error) {
	var result []PR
	page := 1
	for {
		var raw []struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
			User   struct {
				Login     string `json:"login"`
				AvatarURL string `json:"avatar_url"`
			} `json:"user"`
			MergedAt  *time.Time `json:"merged_at"`
			Additions int        `json:"additions"`
			Deletions int        `json:"deletions"`
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
			if r.MergedAt == nil {
				continue
			}
			if r.MergedAt.Before(since) {
				done = true
				break
			}
			pr := PR{
				Number:    r.Number,
				Title:     r.Title,
				Author:    r.User.Login,
				AvatarURL: r.User.AvatarURL,
				MergedAt:  *r.MergedAt,
				Additions: r.Additions,
				Deletions: r.Deletions,
			}
			commits, _ := c.GetPRCommits(ctx, owner, repo, r.Number)
			pr.Commits = commits
			result = append(result, pr)
		}
		if done {
			break
		}
		page++
	}
	return result, nil
}

func (c *Client) GetPRCommits(ctx context.Context, owner, repo string, prNumber int) ([]Commit, error) {
	var raw []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
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
		})
	}
	return commits, nil
}
