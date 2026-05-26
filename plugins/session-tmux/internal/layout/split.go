package layout

// splitPercent computes what percentage of a pane to give to the "remaining" siblings
// so the current pane keeps currentFlex/(currentFlex+remainingFlex) of the space.
//
// With split-window -p N, the NEW pane receives N% and the target keeps (100-N)%.
// Clamped to [1, 99] to avoid degenerate zero-size panes.
func splitPercent(currentFlex, remainingFlex int) int {
	total := currentFlex + remainingFlex
	if total <= 0 {
		return 50
	}

	pct := remainingFlex * 100 / total

	if pct < 1 {
		return 1
	}

	if pct > 99 {
		return 99
	}

	return pct
}

// sumFlex returns the sum of effective flex weights for a slice of panes.
func sumFlex(panes []Pane) int {
	total := 0
	for i := range panes {
		total += panes[i].effectiveFlex()
	}

	return total
}
