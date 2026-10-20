package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Organization identifies a GitHub organization in contribution reports.
type Organization struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// ContributionDay contains one day from GitHub's contribution calendar.
type ContributionDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
	Level string `json:"level"`
	Color string `json:"color"`
}

// ContributionWeek contains the contribution days returned for one calendar week.
type ContributionWeek struct {
	Days []ContributionDay `json:"days"`
}

// ContributionBreakdown contains GitHub contribution counts by activity type.
type ContributionBreakdown struct {
	Commits      int `json:"commits"`
	PullRequests int `json:"pull_requests"`
	Issues       int `json:"issues"`
	CodeReviews  int `json:"code_reviews"`
}

// YearContributions contains one organization's contribution data for one year.
type YearContributions struct {
	Year               int                   `json:"year"`
	TotalContributions int                   `json:"total_contributions"`
	Breakdown          ContributionBreakdown `json:"breakdown"`
	Weeks              []ContributionWeek    `json:"weeks"`
}

// ContributionReport combines all non-empty years in the requested range.
type ContributionReport struct {
	User               string              `json:"user"`
	Organization       Organization        `json:"organization"`
	FromYear           int                 `json:"from_year"`
	ToYear             int                 `json:"to_year"`
	TotalContributions int                 `json:"total_contributions"`
	Years              []YearContributions `json:"years"`
	FetchedAt          time.Time           `json:"fetched_at"`
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

func (c *Client) queryGraphQL(ctx context.Context, query string, variables map[string]any, target any) (err error) {
	if c.token == "" {
		return errors.New("GitHub token is required for contribution data")
	}
	body, err := json.Marshal(graphQLRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("encode GraphQL request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.graphqlURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create GraphQL request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call GitHub GraphQL API: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close GraphQL response: %w", closeErr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return fmt.Errorf("GitHub GraphQL API returned %s: %s", resp.Status, strings.TrimSpace(string(message)))
	}
	var result struct {
		Data   json.RawMessage `json:"data"`
		Errors []graphQLError  `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode GraphQL response: %w", err)
	}
	if len(result.Errors) > 0 {
		messages := make([]error, 0, len(result.Errors))
		for _, graphErr := range result.Errors {
			messages = append(messages, errors.New(graphErr.Message))
		}
		return fmt.Errorf("GitHub GraphQL query failed: %w", errors.Join(messages...))
	}
	if err := json.Unmarshal(result.Data, target); err != nil {
		return fmt.Errorf("decode GraphQL data: %w", err)
	}
	return nil
}

// ViewerLogin returns the login associated with the configured GitHub token.
func (c *Client) ViewerLogin(ctx context.Context) (string, error) {
	const query = `query { viewer { login } }`
	var data struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
	}
	if err := c.queryGraphQL(ctx, query, nil, &data); err != nil {
		return "", err
	}
	if data.Viewer.Login == "" {
		return "", errors.New("GitHub did not return the viewer login")
	}
	return data.Viewer.Login, nil
}

func (c *Client) organization(ctx context.Context, login string) (Organization, error) {
	const query = `query($login: String!) {
  organization(login: $login) { id login name avatarUrl(size: 40) }
}`
	var data struct {
		Organization *Organization `json:"organization"`
	}
	if err := c.queryGraphQL(ctx, query, map[string]any{"login": login}, &data); err != nil {
		return Organization{}, err
	}
	if data.Organization == nil {
		return Organization{}, fmt.Errorf("GitHub organization %q was not found or is not visible", login)
	}
	return *data.Organization, nil
}

// ContributionOrganizations returns organizations where a user contributed in the requested years.
func (c *Client) ContributionOrganizations(ctx context.Context, user string, fromYear, toYear int) (string, []Organization, error) {
	if fromYear < 2008 || toYear < fromYear || toYear > time.Now().UTC().Year() || toYear-fromYear > 20 {
		return "", nil, fmt.Errorf("invalid contribution year range %d-%d", fromYear, toYear)
	}
	if user == "" {
		var err error
		user, err = c.ViewerLogin(ctx)
		if err != nil {
			return "", nil, err
		}
	}
	organizations := make(map[string]Organization)
	for year := fromYear; year <= toYear; year++ {
		yearly, err := c.contributionOrganizationsForYear(ctx, user, year)
		if err != nil {
			return "", nil, fmt.Errorf("get %d contribution organizations: %w", year, err)
		}
		for _, organization := range yearly {
			organizations[strings.ToLower(organization.Login)] = organization
		}
	}
	result := make([]Organization, 0, len(organizations))
	for _, organization := range organizations {
		result = append(result, organization)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Login) < strings.ToLower(result[j].Login)
	})
	return user, result, nil
}

func (c *Client) contributionOrganizationsForYear(ctx context.Context, user string, year int) ([]Organization, error) {
	const query = `query($login: String!, $from: DateTime!, $to: DateTime!) {
  user(login: $login) {
    contributionsCollection(from: $from, to: $to) {
      commitContributionsByRepository(maxRepositories: 100) { repository { owner { __typename id login avatarUrl(size: 40) ... on Organization { name } } } }
      issueContributionsByRepository(maxRepositories: 100) { repository { owner { __typename id login avatarUrl(size: 40) ... on Organization { name } } } }
      pullRequestContributionsByRepository(maxRepositories: 100) { repository { owner { __typename id login avatarUrl(size: 40) ... on Organization { name } } } }
      pullRequestReviewContributionsByRepository(maxRepositories: 100) { repository { owner { __typename id login avatarUrl(size: 40) ... on Organization { name } } } }
    }
  }
}`
	type repositoryContribution struct {
		Repository struct {
			Owner struct {
				Type string `json:"__typename"`
				Organization
			} `json:"owner"`
		} `json:"repository"`
	}
	type collection struct {
		Commits []repositoryContribution `json:"commitContributionsByRepository"`
		Issues  []repositoryContribution `json:"issueContributionsByRepository"`
		Pulls   []repositoryContribution `json:"pullRequestContributionsByRepository"`
		Reviews []repositoryContribution `json:"pullRequestReviewContributionsByRepository"`
	}
	var data struct {
		User *struct {
			Collection collection `json:"contributionsCollection"`
		} `json:"user"`
	}
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, time.December, 31, 23, 59, 59, 0, time.UTC)
	if err := c.queryGraphQL(ctx, query, map[string]any{
		"login": user, "from": from.Format(time.RFC3339), "to": to.Format(time.RFC3339),
	}, &data); err != nil {
		return nil, err
	}
	if data.User == nil {
		return nil, fmt.Errorf("GitHub user %q was not found", user)
	}
	organizations := make(map[string]Organization)
	groups := [][]repositoryContribution{
		data.User.Collection.Commits,
		data.User.Collection.Issues,
		data.User.Collection.Pulls,
		data.User.Collection.Reviews,
	}
	for _, group := range groups {
		for _, contribution := range group {
			owner := contribution.Repository.Owner
			if owner.Type == "Organization" && owner.Login != "" {
				organizations[strings.ToLower(owner.Login)] = owner.Organization
			}
		}
	}
	result := make([]Organization, 0, len(organizations))
	for _, organization := range organizations {
		result = append(result, organization)
	}
	return result, nil
}

// Contributions returns all non-empty yearly contribution calendars for one organization.
func (c *Client) Contributions(ctx context.Context, user, organizationLogin string, fromYear, toYear int) (ContributionReport, error) {
	if fromYear < 2008 || toYear < fromYear || toYear > time.Now().UTC().Year() {
		return ContributionReport{}, fmt.Errorf("invalid contribution year range %d-%d", fromYear, toYear)
	}
	if user == "" {
		var err error
		user, err = c.ViewerLogin(ctx)
		if err != nil {
			return ContributionReport{}, err
		}
	}
	organization := Organization{Login: "all", Name: "All contributions"}
	if organizationLogin != "" {
		var err error
		organization, err = c.organization(ctx, organizationLogin)
		if err != nil {
			return ContributionReport{}, err
		}
	}

	report := ContributionReport{
		User:         user,
		Organization: organization,
		FromYear:     fromYear,
		ToYear:       toYear,
		Years:        make([]YearContributions, 0, toYear-fromYear+1),
		FetchedAt:    time.Now().UTC(),
	}
	for year := toYear; year >= fromYear; year-- {
		yearly, err := c.yearContributions(ctx, user, organization.ID, year)
		if err != nil {
			return ContributionReport{}, fmt.Errorf("get %d contributions: %w", year, err)
		}
		if yearly.TotalContributions == 0 {
			continue
		}
		report.TotalContributions += yearly.TotalContributions
		report.Years = append(report.Years, yearly)
	}
	return report, nil
}

func (c *Client) yearContributions(ctx context.Context, user, organizationID string, year int) (YearContributions, error) {
	const query = `query($login: String!, $organizationID: ID, $from: DateTime!, $to: DateTime!) {
  user(login: $login) {
    contributionsCollection(from: $from, to: $to, organizationID: $organizationID) {
      totalCommitContributions
      totalIssueContributions
      totalPullRequestContributions
      totalPullRequestReviewContributions
      contributionCalendar {
        totalContributions
        weeks { contributionDays { date contributionCount contributionLevel color } }
      }
    }
  }
}`
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, time.December, 31, 23, 59, 59, 0, time.UTC)
	var data struct {
		User *struct {
			Collection struct {
				TotalCommits int `json:"totalCommitContributions"`
				TotalIssues  int `json:"totalIssueContributions"`
				TotalPulls   int `json:"totalPullRequestContributions"`
				TotalReviews int `json:"totalPullRequestReviewContributions"`
				Calendar     struct {
					Total int `json:"totalContributions"`
					Weeks []struct {
						Days []struct {
							Date  string `json:"date"`
							Count int    `json:"contributionCount"`
							Level string `json:"contributionLevel"`
							Color string `json:"color"`
						} `json:"contributionDays"`
					} `json:"weeks"`
				} `json:"contributionCalendar"`
			} `json:"contributionsCollection"`
		} `json:"user"`
	}
	var organizationVariable any
	if organizationID != "" {
		organizationVariable = organizationID
	}
	if err := c.queryGraphQL(ctx, query, map[string]any{
		"login": user, "organizationID": organizationVariable,
		"from": from.Format(time.RFC3339), "to": to.Format(time.RFC3339),
	}, &data); err != nil {
		return YearContributions{}, err
	}
	if data.User == nil {
		return YearContributions{}, fmt.Errorf("GitHub user %q was not found", user)
	}
	collection := data.User.Collection
	yearly := YearContributions{
		Year:               year,
		TotalContributions: collection.Calendar.Total,
		Breakdown: ContributionBreakdown{
			Commits: collection.TotalCommits, PullRequests: collection.TotalPulls,
			Issues: collection.TotalIssues, CodeReviews: collection.TotalReviews,
		},
		Weeks: make([]ContributionWeek, 0, len(collection.Calendar.Weeks)),
	}
	for _, apiWeek := range collection.Calendar.Weeks {
		week := ContributionWeek{Days: make([]ContributionDay, 0, len(apiWeek.Days))}
		for _, apiDay := range apiWeek.Days {
			week.Days = append(week.Days, ContributionDay{
				Date: apiDay.Date, Count: apiDay.Count, Level: apiDay.Level, Color: apiDay.Color,
			})
		}
		yearly.Weeks = append(yearly.Weeks, week)
	}
	return yearly, nil
}
