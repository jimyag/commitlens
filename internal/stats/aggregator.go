package stats

import (
	"fmt"
	"strings"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
	"github.com/jimyag/commitlens/internal/config"
)

func Aggregate(raw *cache.RawData, cfg *config.Config) *cache.StatsData {
	result := &cache.StatsData{
		Repo:         raw.Repo,
		ComputedAt:   time.Now().UTC(),
		Contributors: make(map[string]*cache.ContributorStats),
		Weekly:       make(map[string]*cache.WeeklyEntry),
	}

	// Build a lookup map for aliases
	aliasMap := make(map[string]string)
	if cfg != nil {
		for canonical, aliases := range cfg.UserMap {
			for _, alias := range aliases {
				aliasMap[strings.ToLower(alias)] = canonical
			}
			// Also ensure canonical name itself maps to itself if it appeared as an alias elsewhere
			aliasMap[strings.ToLower(canonical)] = canonical
		}
	}

	resolveName := func(name string) string {
		if canonical, ok := aliasMap[strings.ToLower(name)]; ok {
			return canonical
		}
		return name
	}

	for _, commit := range raw.Commits {
		participants := commit.Participants
		if len(participants) == 0 {
			participants = []string{commit.Author}
		}

		// Map to canonical names and deduplicate within this commit.
		// Use a temporary map to ensure each canonical person is only counted ONCE per commit.
		uniqueInCommit := make(map[string]struct{})
		for _, p := range participants {
			uniqueInCommit[resolveName(p)] = struct{}{}
		}

		for login := range uniqueInCommit {
			avatar := "" // Avatar logic remains minimal for now.
			c := getOrCreate(result.Contributors, login, avatar)
			c.CommitCount++
			c.Additions += commit.Additions
			c.Deletions += commit.Deletions
		}

		week := WeekKey(commit.Date)
		w := getOrCreateWeek(result.Weekly, week)
		w.TotalCommits++
		w.TotalAdditions += commit.Additions
		w.TotalDeletions += commit.Deletions

		for login := range uniqueInCommit {
			cw := getOrCreateContributorWeekly(w.Contributors, login)
			cw.Commits++
			cw.Additions += commit.Additions
			cw.Deletions += commit.Deletions
		}
	}

	return result
}

func getOrCreate(m map[string]*cache.ContributorStats, login, avatarURL string) *cache.ContributorStats {
	if v, ok := m[login]; ok {
		return v
	}
	v := &cache.ContributorStats{Login: login, AvatarURL: avatarURL}
	m[login] = v
	return v
}

func getOrCreateWeek(m map[string]*cache.WeeklyEntry, key string) *cache.WeeklyEntry {
	if v, ok := m[key]; ok {
		return v
	}
	v := &cache.WeeklyEntry{Contributors: make(map[string]*cache.ContributorWeeklyStats)}
	m[key] = v
	return v
}

func getOrCreateContributorWeekly(m map[string]*cache.ContributorWeeklyStats, login string) *cache.ContributorWeeklyStats {
	if v, ok := m[login]; ok {
		return v
	}
	v := &cache.ContributorWeeklyStats{}
	m[login] = v
	return v
}

func WeekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

func MonthKey(t time.Time) string {
	return t.Format("2006-01")
}

func QuarterKey(t time.Time) string {
	q := (int(t.Month())-1)/3 + 1
	return fmt.Sprintf("%d-Q%d", t.Year(), q)
}

func YearKey(t time.Time) string {
	return fmt.Sprintf("%d", t.Year())
}
