package stats

import (
	"fmt"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
)

func Aggregate(raw *cache.RawData) *cache.StatsData {
	result := &cache.StatsData{
		Repo:         raw.Repo,
		ComputedAt:   time.Now().UTC(),
		Contributors: make(map[string]*cache.ContributorStats),
		Weekly:       make(map[string]*cache.WeeklyEntry),
	}

	for _, pr := range raw.PRs {
		participants := PRParticipants(&pr)
		nc := len(pr.Commits)
		// 主作者 + Co-authored-by 合著者；同 PR 内每人只计 1 次。PR/提交/增删行均计给每位参与者（协作 PR 的代码量在多人上可重复计）
		for _, login := range participants {
			avatar := ""
			if login == pr.Author {
				avatar = pr.AvatarURL
			}
			c := getOrCreate(result.Contributors, login, avatar)
			c.PRCount++
			c.CommitCount += nc
			c.Additions += pr.Additions
			c.Deletions += pr.Deletions
		}

		week := WeekKey(pr.MergedAt)
		w := getOrCreateWeek(result.Weekly, week)
		w.TotalPRs++
		for _, login := range participants {
			w.Contributors[login]++
		}
	}

	for _, commit := range raw.DirectCommits {
		// 直接提交，作者仅有一人
		login := commit.Author
		c := getOrCreate(result.Contributors, login, "")
		c.PRCount++ // 将 Direct Commit 计为一次有效的 Submission 从而和 PR 一同显示
		c.CommitCount++
		c.Additions += commit.Additions
		c.Deletions += commit.Deletions

		week := WeekKey(commit.Date)
		w := getOrCreateWeek(result.Weekly, week)
		w.TotalPRs++ // 将 Direct Commit 计入总 Submission 趋势
		w.Contributors[login]++
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
	v := &cache.WeeklyEntry{Contributors: make(map[string]int)}
	m[key] = v
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
