package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

const (
	zoneBody              = "body"
	zoneRepoFilter        = "filter-repo"
	zoneRepoMenu          = "menu-repo"
	zoneGranularityFilter = "filter-granularity"
	zoneGranularityPrev   = "filter-granularity-prev"
	zoneGranularityNext   = "filter-granularity-next"
	zoneLoginFilter       = "filter-login"
	zoneLoginMenu         = "menu-login"
	zoneCommitPeriodPrev  = "commit-period-prev"
	zoneCommitPeriodNext  = "commit-period-next"
	zoneTrendChart        = "trend-chart"
	mouseListScrollStep   = 3
)

func tabZoneID(index int) string {
	return fmt.Sprintf("tab-%d", index)
}

func repoOptionZoneID(index int) string {
	return fmt.Sprintf("repo-option-%d", index)
}

func loginOptionZoneID(index int) string {
	return fmt.Sprintf("login-option-%d", index)
}

func repoRowZoneID(index int) string {
	return fmt.Sprintf("repo-row-%d", index)
}

func commitRowZoneID(index int) string {
	return fmt.Sprintf("commit-row-%d", index)
}

func (a *App) markZone(id, content string) string {
	if a.zones == nil {
		return content
	}
	return a.zones.Mark(id, content)
}

func (a *App) inZone(id string, msg tea.MouseMsg) bool {
	if a.zones == nil {
		return false
	}
	return a.zones.Get(id).InBounds(msg)
}

func (a *App) handleMouseClick(msg tea.MouseClickMsg) {
	for i := viewSummary; i <= viewCommitList; i++ {
		if a.inZone(tabZoneID(int(i)), msg) {
			a.mode = i
			a.globalRepoExpanded = false
			a.globalLoginExpanded = false
			a.globalLoginSearch = ""
			if a.mode == viewCommitList {
				a.refreshCommitList()
			}
			return
		}
	}

	if a.inZone(zoneGranularityPrev, msg) {
		a.changeGranularity(-1)
		return
	}
	if a.inZone(zoneGranularityNext, msg) {
		a.changeGranularity(1)
		return
	}
	if a.inZone(zoneCommitPeriodPrev, msg) {
		a.changeCommitListPeriod(-1)
		return
	}
	if a.inZone(zoneCommitPeriodNext, msg) {
		a.changeCommitListPeriod(1)
		return
	}

	if a.globalRepoExpanded {
		visible := a.visibleRepoIndices()
		for _, repoIdx := range visible {
			if a.inZone(repoOptionZoneID(repoIdx), msg) {
				for cursor, idx := range visible {
					if idx == repoIdx {
						a.globalRepoCursor = cursor
						break
					}
				}
				a.toggleGlobalRepo(repoIdx)
				return
			}
		}
	}

	if a.globalLoginExpanded {
		filtered, start, end := a.visibleLoginOptions()
		for i := start; i < end; i++ {
			if a.inZone(loginOptionZoneID(i), msg) {
				a.globalLoginCursor = i
				a.toggleGlobalLogin(filtered[i])
				return
			}
		}
	}

	if a.inZone(zoneRepoFilter, msg) {
		a.globalFocus = 0
		a.globalRepoExpanded = !a.globalRepoExpanded
		a.globalLoginExpanded = false
		a.globalLoginSearch = ""
		if a.globalRepoExpanded {
			a.globalRepoCursor = 0
		}
		return
	}
	if a.inZone(zoneGranularityFilter, msg) {
		a.globalFocus = 1
		a.globalRepoExpanded = false
		a.globalLoginExpanded = false
		a.globalLoginSearch = ""
		return
	}
	if a.inZone(zoneLoginFilter, msg) {
		a.globalFocus = 2
		a.globalLoginExpanded = !a.globalLoginExpanded
		a.globalRepoExpanded = false
		a.globalLoginSearch = ""
		if a.globalLoginExpanded {
			a.globalLoginCursor = 0
		}
		return
	}

	if a.mode == viewRepo {
		for i := range a.repoNames {
			if a.inZone(repoRowZoneID(i), msg) {
				a.globalFocus = 3
				a.repoViewSelected = i
				return
			}
		}
	}

	if a.mode == viewCommitList {
		start, end := a.commitListVisibleRange()
		for i := start; i < end; i++ {
			if a.inZone(commitRowZoneID(i), msg) {
				a.globalFocus = 3
				a.commitListFocus = 1
				a.commitListCursor = i
				return
			}
		}
	}

	if (a.mode == viewTrend || a.mode == viewLinesTrend) && a.inZone(zoneTrendChart, msg) {
		x, _ := a.zones.Get(zoneTrendChart).Pos(msg)
		periods := sortedPeriodKeys(aggregatePeriods(a))
		index := trendPeriodIndexAt(len(periods), a.width, a.trendHScroll, x)
		if index >= 0 {
			a.globalFocus = 3
			a.trendPeriodCursor = index
			a.openCommitListFromTrend()
		}
		return
	}

	if a.inZone(zoneBody, msg) {
		a.globalFocus = 3
		return
	}

	a.globalRepoExpanded = false
	a.globalLoginExpanded = false
	a.globalLoginSearch = ""
}

func (a *App) handleMouseWheel(msg tea.MouseWheelMsg) {
	direction := mouseWheelDirection(msg.Button)
	if direction == 0 {
		return
	}

	if a.globalRepoExpanded && a.inZone(zoneRepoMenu, msg) {
		visible := a.visibleRepoIndices()
		a.globalRepoCursor = clamp(a.globalRepoCursor+direction, 0, len(visible)-1)
		return
	}
	if a.globalLoginExpanded && a.inZone(zoneLoginMenu, msg) {
		filtered := a.filteredLogins()
		a.globalLoginCursor = clamp(a.globalLoginCursor+direction, 0, len(filtered)-1)
		return
	}
	if a.inZone(zoneGranularityFilter, msg) {
		a.changeGranularity(direction)
		return
	}
	if (a.mode == viewTrend || a.mode == viewLinesTrend) && a.inZone(zoneTrendChart, msg) {
		a.trendHScroll += direction * trendHScrollStep
		if a.trendHScroll < 0 {
			a.trendHScroll = 0
		}
		return
	}
	if a.mode == viewRepo && a.inZone(zoneBody, msg) {
		a.globalFocus = 3
		a.repoViewSelected = clamp(a.repoViewSelected+direction, 0, len(a.repoNames)-1)
		return
	}
	if a.mode == viewCommitList && a.inZone(zoneBody, msg) {
		a.globalFocus = 3
		a.commitListFocus = 1
		a.commitListCursor = clamp(a.commitListCursor+direction*mouseListScrollStep, 0, len(a.commitList)-1)
	}
}

func mouseWheelDirection(button tea.MouseButton) int {
	switch button {
	case tea.MouseWheelUp, tea.MouseWheelLeft:
		return -1
	case tea.MouseWheelDown, tea.MouseWheelRight:
		return 1
	default:
		return 0
	}
}

func (a *App) changeGranularity(delta int) {
	next := clamp(a.globalGranularity+delta, 0, 3)
	if next == a.globalGranularity {
		return
	}
	a.globalFocus = 1
	a.globalGranularity = next
	a.trendPeriodCursor = 0
	a.trendHScroll = 0
	a.onGlobalFilterChange()
}

func (a *App) changeCommitListPeriod(delta int) {
	periods := a.availableCommitListPeriods()
	next := clamp(a.commitListPeriodIdx+delta, 0, len(periods)-1)
	if next == a.commitListPeriodIdx {
		return
	}
	a.globalFocus = 3
	a.commitListFocus = 0
	a.commitListPeriodIdx = next
	a.refreshCommitList()
}

func (a *App) visibleLoginOptions() ([]string, int, int) {
	filtered := a.filteredLogins()
	start := 0
	if a.globalLoginCursor >= 10 {
		start = a.globalLoginCursor - 9
	}
	end := start + 10
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered, start, end
}

func (a *App) commitListVisibleRange() (int, int) {
	visibleHeight := a.height - 10
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	start := 0
	if a.commitListCursor >= visibleHeight {
		start = a.commitListCursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(a.commitList) {
		end = len(a.commitList)
	}
	return start, end
}

func trendPeriodIndexAt(periodCount, viewWidth, scrollOffset, x int) int {
	if periodCount <= 0 || x < 0 {
		return -1
	}
	chartWidth := trendChartCanvasW(periodCount, viewWidth)
	gap := pickBarGap(periodCount, chartWidth)
	barWidth := (chartWidth - (periodCount-1)*gap) / periodCount
	if barWidth < 1 {
		barWidth = 1
	}
	logicalX := x + scrollOffset
	slotWidth := barWidth + gap
	index := logicalX / slotWidth
	if index < 0 || index >= periodCount || logicalX%slotWidth >= barWidth {
		return -1
	}
	return index
}

func clamp(value, minValue, maxValue int) int {
	if maxValue < minValue {
		return minValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
