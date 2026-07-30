package stats

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jimyag/commitlens/internal/cache"
)

type Granularity string

const (
	GranularityWeek    Granularity = "week"
	GranularityMonth   Granularity = "month"
	GranularityQuarter Granularity = "quarter"
	GranularityYear    Granularity = "year"
)

type ComparisonMetrics struct {
	Commits   int `json:"commits"`
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
}

type ComparisonPoint struct {
	Key       string            `json:"key"`
	Label     string            `json:"label"`
	Available bool              `json:"available"`
	Complete  bool              `json:"complete"`
	Metrics   ComparisonMetrics `json:"metrics"`
}

type PeriodComparison struct {
	Anchor     ComparisonPoint   `json:"anchor"`
	Previous   ComparisonPoint   `json:"previous"`
	Historical []ComparisonPoint `json:"historical"`
}

type ComparisonSet struct {
	Granularity Granularity                 `json:"granularity"`
	Periods     []string                    `json:"periods"`
	Comparisons map[string]PeriodComparison `json:"comparisons"`
}

type ComparisonOptions struct {
	Granularity Granularity
	Logins      []string
	Location    *time.Location
	Now         time.Time
}

type periodWindow struct {
	key   string
	label string
	from  time.Time
	to    time.Time
	valid bool
}

func BuildComparisons(data []*cache.StatsData, options ComparisonOptions) ComparisonSet {
	location := options.Location
	if location == nil {
		location = time.Local
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	now = now.In(location)

	periodSet := make(map[string]struct{})
	var coverageStart time.Time
	for _, repository := range data {
		for day := range repository.Daily {
			date, err := time.ParseInLocation("2006-01-02", day, location)
			if err != nil {
				continue
			}
			periodSet[periodKey(date, options.Granularity)] = struct{}{}
			if coverageStart.IsZero() || date.Before(coverageStart) {
				coverageStart = date
			}
		}
	}

	periods := make([]string, 0, len(periodSet))
	for period := range periodSet {
		periods = append(periods, period)
	}
	sort.Strings(periods)

	result := ComparisonSet{
		Granularity: options.Granularity,
		Periods:     periods,
		Comparisons: make(map[string]PeriodComparison, len(periods)),
	}
	for _, period := range periods {
		result.Comparisons[period] = comparePeriod(data, options, period, coverageStart, now, location)
	}
	return result
}

func ComparePeriod(data []*cache.StatsData, options ComparisonOptions, anchorKey string) PeriodComparison {
	location := options.Location
	if location == nil {
		location = time.Local
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	now = now.In(location)

	var coverageStart time.Time
	for _, repository := range data {
		for day := range repository.Daily {
			date, err := time.ParseInLocation("2006-01-02", day, location)
			if err == nil && (coverageStart.IsZero() || date.Before(coverageStart)) {
				coverageStart = date
			}
		}
	}
	return comparePeriod(data, options, anchorKey, coverageStart, now, location)
}

func comparePeriod(
	data []*cache.StatsData,
	options ComparisonOptions,
	anchorKey string,
	coverageStart time.Time,
	now time.Time,
	location *time.Location,
) PeriodComparison {
	anchor := windowForKey(anchorKey, options.Granularity, location)
	elapsedEnd := anchor.to
	if now.Before(elapsedEnd) {
		elapsedEnd = now
	}
	elapsed := elapsedEnd.Sub(anchor.from)
	if elapsed < 0 {
		elapsed = 0
	}

	point := func(window periodWindow) ComparisonPoint {
		if !window.valid {
			return ComparisonPoint{Key: window.key, Label: window.label}
		}
		end := window.from.Add(elapsed)
		if end.After(window.to) {
			end = window.to
		}
		available := !coverageStart.IsZero() && window.to.After(coverageStart) && !window.from.After(now)
		value := ComparisonMetrics{}
		if available {
			value = metricsInRange(data, options.Logins, window.from, end, location)
		}
		return ComparisonPoint{
			Key:       window.key,
			Label:     window.label,
			Available: available,
			Complete:  !window.to.After(now),
			Metrics:   value,
		}
	}

	historyWindows := historicalWindows(anchor, options.Granularity, location)
	history := make([]ComparisonPoint, 0, len(historyWindows))
	for _, window := range historyWindows {
		history = append(history, point(window))
	}

	return PeriodComparison{
		Anchor:     point(anchor),
		Previous:   point(previousWindow(anchor, options.Granularity, location)),
		Historical: history,
	}
}

func metricsInRange(
	data []*cache.StatsData,
	logins []string,
	from time.Time,
	to time.Time,
	location *time.Location,
) ComparisonMetrics {
	selected := make(map[string]struct{}, len(logins))
	for _, login := range logins {
		if login != "" {
			selected[login] = struct{}{}
		}
	}

	var result ComparisonMetrics
	for _, repository := range data {
		for day, entry := range repository.Daily {
			date, err := time.ParseInLocation("2006-01-02", day, location)
			if err != nil || date.Before(from) || !date.Before(to) {
				continue
			}
			if len(selected) == 0 {
				result.Commits += entry.TotalCommits
				result.Additions += entry.TotalAdditions
				result.Deletions += entry.TotalDeletions
				continue
			}
			for login := range selected {
				contributor := entry.Contributors[login]
				if contributor == nil {
					continue
				}
				result.Commits += contributor.Commits
				result.Additions += contributor.Additions
				result.Deletions += contributor.Deletions
			}
		}
	}
	return result
}

func windowForKey(key string, granularity Granularity, location *time.Location) periodWindow {
	switch granularity {
	case GranularityWeek:
		var year, week int
		if _, err := fmt.Sscanf(key, "%d-W%d", &year, &week); err != nil {
			return periodWindow{key: key, label: key}
		}
		jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, location)
		weekday := int(jan4.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		from := jan4.AddDate(0, 0, -weekday+1+(week-1)*7)
		return periodWindow{key: key, label: key, from: from, to: from.AddDate(0, 0, 7), valid: true}
	case GranularityMonth:
		parts := strings.Split(key, "-")
		if len(parts) != 2 {
			return periodWindow{key: key, label: key}
		}
		year, yearErr := strconv.Atoi(parts[0])
		month, monthErr := strconv.Atoi(parts[1])
		if yearErr != nil || monthErr != nil || month < 1 || month > 12 {
			return periodWindow{key: key, label: key}
		}
		from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, location)
		return periodWindow{key: key, label: key, from: from, to: from.AddDate(0, 1, 0), valid: true}
	case GranularityQuarter:
		var year, quarter int
		if _, err := fmt.Sscanf(key, "%d-Q%d", &year, &quarter); err != nil || quarter < 1 || quarter > 4 {
			return periodWindow{key: key, label: key}
		}
		from := time.Date(year, time.Month((quarter-1)*3+1), 1, 0, 0, 0, 0, location)
		return periodWindow{key: key, label: key, from: from, to: from.AddDate(0, 3, 0), valid: true}
	case GranularityYear:
		year, err := strconv.Atoi(key)
		if err != nil {
			return periodWindow{key: key, label: key}
		}
		from := time.Date(year, time.January, 1, 0, 0, 0, 0, location)
		return periodWindow{key: key, label: key, from: from, to: from.AddDate(1, 0, 0), valid: true}
	default:
		return periodWindow{key: key, label: key}
	}
}

func previousWindow(anchor periodWindow, granularity Granularity, location *time.Location) periodWindow {
	var from time.Time
	switch granularity {
	case GranularityWeek:
		from = anchor.from.AddDate(0, 0, -7)
	case GranularityMonth:
		from = anchor.from.AddDate(0, -1, 0)
	case GranularityQuarter:
		from = anchor.from.AddDate(0, -3, 0)
	case GranularityYear:
		from = anchor.from.AddDate(-1, 0, 0)
	default:
		return periodWindow{}
	}
	key := periodKey(from, granularity)
	return windowForKey(key, granularity, location)
}

func historicalWindows(anchor periodWindow, granularity Granularity, location *time.Location) []periodWindow {
	result := make([]periodWindow, 0, 3)
	switch granularity {
	case GranularityWeek:
		ordinal := (anchor.from.Day()-1)/7 + 1
		for offset := 1; offset <= 3; offset++ {
			monthStart := time.Date(anchor.from.Year(), anchor.from.Month(), 1, 0, 0, 0, 0, location).AddDate(0, -offset, 0)
			weekday := int(monthStart.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			firstMonday := monthStart.AddDate(0, 0, (8-weekday)%7)
			from := firstMonday.AddDate(0, 0, (ordinal-1)*7)
			label := fmt.Sprintf("%04d-%02d W%d", monthStart.Year(), monthStart.Month(), ordinal)
			if from.Month() != monthStart.Month() {
				result = append(result, periodWindow{key: label, label: label})
				continue
			}
			key := periodKey(from, granularity)
			result = append(result, periodWindow{
				key:   key,
				label: label,
				from:  from,
				to:    from.AddDate(0, 0, 7),
				valid: true,
			})
		}
	case GranularityMonth:
		for offset := 1; offset <= 3; offset++ {
			from := anchor.from.AddDate(-offset, 0, 0)
			result = append(result, windowForKey(periodKey(from, granularity), granularity, location))
		}
	case GranularityQuarter:
		for offset := 1; offset <= 3; offset++ {
			from := anchor.from.AddDate(-offset, 0, 0)
			result = append(result, windowForKey(periodKey(from, granularity), granularity, location))
		}
	case GranularityYear:
		for offset := 2; offset <= 4; offset++ {
			from := anchor.from.AddDate(-offset, 0, 0)
			result = append(result, windowForKey(periodKey(from, granularity), granularity, location))
		}
	}
	return result
}

func periodKey(date time.Time, granularity Granularity) string {
	switch granularity {
	case GranularityWeek:
		return WeekKey(date)
	case GranularityMonth:
		return MonthKey(date)
	case GranularityQuarter:
		return QuarterKey(date)
	case GranularityYear:
		return YearKey(date)
	default:
		return ""
	}
}
