package tui

func (a *App) normalizeState() {
	a.mode = viewMode(clamp(int(a.mode), int(viewSummary), int(viewCommitList)))
	a.globalFocus = clamp(a.globalFocus, 0, 3)
	a.globalGranularity = clamp(a.globalGranularity, 0, 3)
	a.commitListFocus = clamp(a.commitListFocus, 0, 1)

	a.repoViewSelected = clamp(a.repoViewSelected, 0, len(a.repoNames)-1)

	for idx := range a.globalRepoMulti {
		if idx < 0 || idx >= len(a.repoNames) {
			delete(a.globalRepoMulti, idx)
		}
	}
	if a.globalRepoMulti != nil && len(a.globalRepoMulti) == 0 {
		a.globalRepoMulti = nil
	}

	visibleRepos := a.visibleRepoIndices()
	a.globalRepoCursor = clamp(a.globalRepoCursor, 0, len(visibleRepos)-1)

	logins := a.availableGlobalLogins()
	a.globalLoginIdx = clamp(a.globalLoginIdx, 0, len(logins)-1)

	filteredLogins := a.filteredLogins()
	a.globalLoginCursor = clamp(a.globalLoginCursor, 0, len(filteredLogins)-1)

	periods := sortedPeriodKeys(aggregatePeriods(a))
	a.trendPeriodCursor = clamp(a.trendPeriodCursor, 0, len(periods)-1)

	commitPeriods := a.availableCommitListPeriods()
	if a.commitListPeriodIdx < 0 || a.commitListPeriodIdx >= len(commitPeriods) {
		a.commitListPeriodIdx = 0
	}
	a.commitListCursor = clamp(a.commitListCursor, 0, len(a.commitList)-1)
}
