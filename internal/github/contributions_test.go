package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type graphQLRoundTripper func(*http.Request) (*http.Response, error)

func (f graphQLRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type graphQLResponseBody struct {
	io.Reader
	closed bool
	err    error
}

func (b *graphQLResponseBody) Close() error {
	b.closed = true
	return b.err
}

func TestQueryGraphQLResponseCloseError(t *testing.T) {
	closeErr := errors.New("response close failed")
	for _, response := range []string{`{"data":{"viewer":{"login":"octocat"}}}`, `invalid JSON`} {
		t.Run(response, func(t *testing.T) {
			body := &graphQLResponseBody{Reader: strings.NewReader(response), err: closeErr}
			client := NewClient("test-token")
			client.httpClient.Transport = graphQLRoundTripper(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
			})
			_, err := client.ViewerLogin(t.Context())
			if !body.closed {
				t.Fatal("response body was not closed")
			}
			if response == "invalid JSON" {
				if err == nil || !strings.Contains(err.Error(), "decode GraphQL response") {
					t.Fatalf("ViewerLogin() error = %v, want decode error", err)
				}
			} else if !errors.Is(err, closeErr) {
				t.Fatalf("ViewerLogin() error = %v, want close error", err)
			}
		})
	}
}

func TestContributionsUsesViewerAndSkipsEmptyYears(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		var request graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(request.Query, "viewer"):
			_, _ = fmt.Fprint(w, `{"data":{"viewer":{"login":"octocat"}}}`)
		case strings.Contains(request.Query, "organization(login"):
			_, _ = fmt.Fprint(w, `{"data":{"organization":{"id":"O_1","login":"example-org","name":"Example"}}}`)
		default:
			year := 2024
			if strings.HasPrefix(request.Variables["from"].(string), "2023-") {
				year = 2023
			}
			total := 0
			if year == 2024 {
				total = 7
			}
			_, _ = fmt.Fprintf(w, `{"data":{"user":{"contributionsCollection":{"totalCommitContributions":2,"totalIssueContributions":1,"totalPullRequestContributions":3,"totalPullRequestReviewContributions":1,"contributionCalendar":{"totalContributions":%d,"weeks":[{"contributionDays":[{"date":"%d-01-01","contributionCount":%d,"contributionLevel":"FOURTH_QUARTILE","color":"#216e39"}]}]}}}}}`, total, year, total)
		}
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.httpClient = server.Client()
	client.graphqlURL = server.URL
	report, err := client.Contributions(context.Background(), "", "example-org", 2023, 2024)
	if err != nil {
		t.Fatalf("Contributions() error = %v", err)
	}
	if report.User != "octocat" || report.Organization.Login != "example-org" {
		t.Fatalf("unexpected report identity: %#v", report)
	}
	if report.TotalContributions != 7 || len(report.Years) != 1 || report.Years[0].Year != 2024 {
		t.Fatalf("unexpected report years: %#v", report)
	}
	if report.Years[0].Breakdown != (ContributionBreakdown{Commits: 2, PullRequests: 3, Issues: 1, CodeReviews: 1}) {
		t.Fatalf("unexpected breakdown: %#v", report.Years[0].Breakdown)
	}
}

func TestContributionsRejectsInvalidRange(t *testing.T) {
	t.Parallel()
	_, err := NewClient("test-token").Contributions(context.Background(), "octocat", "example-org", 2025, 2024)
	if err == nil || !strings.Contains(err.Error(), "invalid contribution year range") {
		t.Fatalf("Contributions() error = %v", err)
	}
}

func TestContributionsWithoutOrganizationQueriesAllActivity(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if strings.Contains(request.Query, "organization(login") {
			t.Fatal("all contributions must not resolve an organization")
		}
		if request.Variables["organizationID"] != nil {
			t.Fatalf("organizationID = %#v, want nil", request.Variables["organizationID"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"user":{"contributionsCollection":{"totalCommitContributions":4,"totalIssueContributions":1,"totalPullRequestContributions":2,"totalPullRequestReviewContributions":1,"contributionCalendar":{"totalContributions":8,"weeks":[]}}}}}`)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.httpClient = server.Client()
	client.graphqlURL = server.URL
	report, err := client.Contributions(context.Background(), "octocat", "", 2024, 2024)
	if err != nil {
		t.Fatalf("Contributions() error = %v", err)
	}
	if report.Organization.Login != "all" || report.TotalContributions != 8 {
		t.Fatalf("unexpected all-contribution report: %#v", report)
	}
}

func TestContributionOrganizationsCombinesAndSortsYears(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(request.Query, "viewer") {
			_, _ = fmt.Fprint(w, `{"data":{"viewer":{"login":"octocat"}}}`)
			return
		}
		owner := `{"__typename":"Organization","id":"O_2","login":"zeta","name":"Zeta"}`
		if strings.HasPrefix(request.Variables["from"].(string), "2024-") {
			owner = `{"__typename":"Organization","id":"O_1","login":"alpha","name":"Alpha"}`
		}
		_, _ = fmt.Fprintf(w, `{"data":{"user":{"contributionsCollection":{"commitContributionsByRepository":[{"repository":{"owner":%s}}],"issueContributionsByRepository":[],"pullRequestContributionsByRepository":[],"pullRequestReviewContributionsByRepository":[]}}}}`, owner)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.httpClient = server.Client()
	client.graphqlURL = server.URL
	user, organizations, err := client.ContributionOrganizations(context.Background(), "", 2023, 2024)
	if err != nil {
		t.Fatalf("ContributionOrganizations() error = %v", err)
	}
	if user != "octocat" || len(organizations) != 2 || organizations[0].Login != "alpha" || organizations[1].Login != "zeta" {
		t.Fatalf("ContributionOrganizations() = %q, %#v", user, organizations)
	}
}
