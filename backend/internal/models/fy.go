package models

import (
	"fmt"
	"time"
)

// FYWindow is an Indian financial year window: 1 Apr to 31 Mar.
type FYWindow struct {
	Label string    `json:"label"` // e.g. "FY 2026-27"
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// CurrentFY returns the Indian financial year window containing `from`.
func CurrentFY(from time.Time) FYWindow {
	startYear := from.Year()
	if from.Month() < time.April {
		startYear--
	}
	start := time.Date(startYear, time.April, 1, 0, 0, 0, 0, from.Location())
	end := time.Date(startYear+1, time.March, 31, 23, 59, 59, 0, from.Location())
	label := fmt.Sprintf("FY %d-%02d", startYear, (startYear+1)%100)
	return FYWindow{Label: label, Start: start, End: end}
}
