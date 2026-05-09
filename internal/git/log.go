package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var coAuthoredByLine = regexp.MustCompile(`(?mi)^Co-authored-by:\s+(.*?)\s*<.*?>$`)

// GetCommits extracts commits and their stats from the given git directory.
// revRange specifies the range of commits to fetch (e.g., "oldSHA..HEAD").
// If empty, it fetches everything reachable from HEAD (or origin/HEAD in bare clones).
func GetCommits(ctx context.Context, gitDir, revRange string) ([]Commit, error) {
	// COMMIT_BEGIN<NUL>hash<NUL>authorName<NUL>authorEmail<NUL>date<NUL>subject<NUL>body<NUL>COMMIT_HEADER_END
	format := "COMMIT_BEGIN%x00%H%x00%aN%x00%aE%x00%aI%x00%s%x00%b%x00COMMIT_HEADER_END"

	args := []string{"--git-dir=" + gitDir, "log", "--format=" + format, "--numstat"}
	if revRange != "" {
		args = append(args, revRange)
	} else {
		// Use HEAD or origin/master etc. depending on what's available.
		// For bare clones of github, origin/HEAD or specific branches are usually what we want.
		// By default 'git log' uses HEAD.
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	return parseGitLog(out)
}

func parseGitLog(data []byte) ([]Commit, error) {
	var commits []Commit
	parts := bytes.Split(data, []byte("COMMIT_BEGIN\x00"))

	for _, part := range parts {
		if len(bytes.TrimSpace(part)) == 0 {
			continue
		}

		headerAndRest := bytes.SplitN(part, []byte("\x00COMMIT_HEADER_END"), 2)
		if len(headerAndRest) != 2 {
			continue
		}

		headerFields := bytes.SplitN(headerAndRest[0], []byte{0}, 6)
		if len(headerFields) != 6 {
			continue
		}

		hash := string(headerFields[0])
		author := string(headerFields[1])
		email := string(headerFields[2])
		dateStr := string(headerFields[3])
		subject := string(headerFields[4])
		body := string(headerFields[5])

		t, _ := time.Parse(time.RFC3339, dateStr)

		participants := []string{author}
		matches := coAuthoredByLine.FindAllStringSubmatch(body, -1)
		for _, m := range matches {
			if len(m) > 1 {
				coAuthor := strings.TrimSpace(m[1])
				found := false
				for _, p := range participants {
					if p == coAuthor {
						found = true
						break
					}
				}
				if !found {
					participants = append(participants, coAuthor)
				}
			}
		}

		rest := headerAndRest[1]
		additions, deletions := 0, 0
		lines := strings.Split(string(rest), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if fields[0] != "-" {
					if a, err := strconv.Atoi(fields[0]); err == nil {
						additions += a
					}
				}
				if fields[1] != "-" {
					if d, err := strconv.Atoi(fields[1]); err == nil {
						deletions += d
					}
				}
			}
		}

		commits = append(commits, Commit{
			SHA:          hash,
			Author:       author,
			AuthorEmail:  email,
			Participants: participants,
			Message:      subject,
			Date:         t,
			Additions:    additions,
			Deletions:    deletions,
		})
	}

	return commits, nil
}
