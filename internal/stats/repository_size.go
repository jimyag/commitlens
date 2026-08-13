package stats

import (
	"sort"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
	gitrepo "github.com/jimyag/commitlens/internal/git"
)

var repositorySizeGranularities = []Granularity{
	GranularityWeek,
	GranularityMonth,
	GranularityQuarter,
	GranularityYear,
}

// BuildRepositorySize groups default-branch line counts into period-end
// snapshots. Periods without commits carry forward the last known tree size.
func BuildRepositorySize(history []gitrepo.LineCountPoint, location *time.Location, now time.Time) map[string]map[string]cache.RepositorySizePoint {
	result := make(map[string]map[string]cache.RepositorySizePoint, len(repositorySizeGranularities))
	if len(history) == 0 {
		return result
	}
	if location == nil {
		location = time.Local
	}
	if now.IsZero() {
		now = time.Now()
	}

	ordered := append([]gitrepo.LineCountPoint(nil), history...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Date.Before(ordered[j].Date) })
	for _, granularity := range repositorySizeGranularities {
		result[string(granularity)] = buildRepositorySizePeriods(ordered, granularity, location, now.In(location))
	}
	return result
}

func buildRepositorySizePeriods(history []gitrepo.LineCountPoint, granularity Granularity, location *time.Location, now time.Time) map[string]cache.RepositorySizePoint {
	result := make(map[string]cache.RepositorySizePoint)
	periodStart := startOfSizePeriod(history[0].Date.In(location), granularity)
	index := 0
	var latest *gitrepo.LineCountPoint

	for !periodStart.After(now) {
		next := nextSizePeriod(periodStart, granularity)
		limit := next
		if now.Before(limit) {
			limit = now.Add(time.Nanosecond)
		}
		for index < len(history) && history[index].Date.In(location).Before(limit) {
			point := history[index]
			latest = &point
			index++
		}
		if latest != nil {
			result[periodKey(periodStart, granularity)] = cache.RepositorySizePoint{
				Lines:  latest.Lines,
				Commit: latest.SHA,
				Date:   latest.Date,
			}
		}
		periodStart = next
	}
	return result
}

func startOfSizePeriod(value time.Time, granularity Granularity) time.Time {
	year, month, day := value.Date()
	switch granularity {
	case GranularityWeek:
		weekday := (int(value.Weekday()) + 6) % 7
		return time.Date(year, month, day-weekday, 0, 0, 0, 0, value.Location())
	case GranularityMonth:
		return time.Date(year, month, 1, 0, 0, 0, 0, value.Location())
	case GranularityQuarter:
		quarterMonth := time.Month((int(month)-1)/3*3 + 1)
		return time.Date(year, quarterMonth, 1, 0, 0, 0, 0, value.Location())
	default:
		return time.Date(year, time.January, 1, 0, 0, 0, 0, value.Location())
	}
}

func nextSizePeriod(value time.Time, granularity Granularity) time.Time {
	switch granularity {
	case GranularityWeek:
		return value.AddDate(0, 0, 7)
	case GranularityMonth:
		return value.AddDate(0, 1, 0)
	case GranularityQuarter:
		return value.AddDate(0, 3, 0)
	default:
		return value.AddDate(1, 0, 0)
	}
}
