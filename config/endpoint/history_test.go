package endpoint

import (
	"testing"
	"time"
)

func newTestResult(timestamp time.Time, success bool, duration time.Duration) *Result {
	return &Result{
		Timestamp: timestamp,
		Success:   success,
		Duration:  duration,
		ConditionResults: []*ConditionResult{
			{Condition: "[STATUS] == 200", Success: success},
		},
	}
}

func TestNewHistory(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	history := NewHistory(from, to, NumberOfHistoryBins)
	if len(history.Bins) != NumberOfHistoryBins {
		t.Fatalf("expected %d bins, got %d", NumberOfHistoryBins, len(history.Bins))
	}
	if history.Window != time.Hour {
		t.Errorf("expected window of 1h, got %s", history.Window)
	}
	if history.BinWidth() != 72*time.Second {
		t.Errorf("expected bin width of 72s, got %s", history.BinWidth())
	}
	if !history.Bins[0].Start.Equal(from) || !history.Bins[len(history.Bins)-1].End.Equal(to) {
		t.Error("expected bins to span the entire window")
	}
	for i, bin := range history.Bins {
		if bin.State != HistoryStateUnknown {
			t.Errorf("bin %d: expected state %s, got %s", i, HistoryStateUnknown, bin.State)
		}
		if i > 0 && !bin.Start.Equal(history.Bins[i-1].End) {
			t.Errorf("bin %d: expected start to be equal to the end of the previous bin", i)
		}
	}
	if len(NewHistory(from, to, 0).Bins) != 1 {
		t.Error("expected at least one bin")
	}
}

func TestNewHistoryWindow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 12, 34, 56, 0, time.UTC)
	from, to := NewHistoryWindow(now, 24*time.Hour, NumberOfHistoryBins)
	if to.Sub(from) != 24*time.Hour {
		t.Errorf("expected window of 24h, got %s", to.Sub(from))
	}
	if !to.After(now) {
		t.Error("expected the end of the window to be after now")
	}
	binWidth := 24 * time.Hour / NumberOfHistoryBins
	if to.Sub(now) > binWidth {
		t.Errorf("expected the end of the window to be at most one bin after now, got %s", to.Sub(now))
	}
	if to.UnixNano()%int64(binWidth) != 0 {
		t.Error("expected the end of the window to be aligned on a bin boundary")
	}
	// A slightly later call within the same bin must return the same window
	laterFrom, laterTo := NewHistoryWindow(now.Add(time.Second), 24*time.Hour, NumberOfHistoryBins)
	if !laterFrom.Equal(from) || !laterTo.Equal(to) {
		t.Error("expected the window to be stable within a bin")
	}
}

func TestHistory_BinIndex(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	history := NewHistory(from, from.Add(10*time.Minute), 10)
	scenarios := []struct {
		timestamp time.Time
		expected  int
	}{
		{from.Add(-time.Nanosecond), -1},
		{from, 0},
		{from.Add(59 * time.Second), 0},
		{from.Add(time.Minute), 1},
		{from.Add(10*time.Minute - time.Nanosecond), 9},
		{from.Add(10 * time.Minute), -1},
	}
	for _, scenario := range scenarios {
		if index := history.BinIndex(scenario.timestamp); index != scenario.expected {
			t.Errorf("expected index of %s to be %d, got %d", scenario.timestamp, scenario.expected, index)
		}
	}
}

func TestHistory_Finalize(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newHistory := func() *History {
		history := NewHistory(from, from.Add(6*time.Minute), 6)
		history.AddCheck(newTestResult(from.Add(-30*time.Second), true, time.Millisecond)) // before the window
		history.AddCheck(newTestResult(from.Add(time.Minute), true, 100*time.Millisecond))
		history.AddCheck(newTestResult(from.Add(time.Minute+30*time.Second), true, 300*time.Millisecond))
		history.AddCheck(newTestResult(from.Add(2*time.Minute), false, 0))
		history.AddCheck(newTestResult(from.Add(3*time.Minute), true, 0))
		history.AddCheck(newTestResult(from.Add(3*time.Minute+30*time.Second), false, 0))
		history.AddCheck(newTestResult(from.Add(time.Hour), false, 0)) // after the window
		return history
	}
	t.Run("no-carry-over-limit", func(t *testing.T) {
		history := newHistory()
		history.Finalize(0)
		expected := []HistoryState{HistoryStateUp, HistoryStateUp, HistoryStateDown, HistoryStateDegraded, HistoryStateDown, HistoryStateDown}
		for i, bin := range history.Bins {
			if bin.State != expected[i] {
				t.Errorf("bin %d: expected state %s, got %s", i, expected[i], bin.State)
			}
		}
		if history.Bins[1].AverageDuration != 200*time.Millisecond {
			t.Errorf("expected average duration of 200ms, got %s", history.Bins[1].AverageDuration)
		}
		if history.Bins[1].MinimumDuration != 100*time.Millisecond || history.Bins[1].MaximumDuration != 300*time.Millisecond {
			t.Errorf("expected min=100ms and max=300ms, got min=%s and max=%s", history.Bins[1].MinimumDuration, history.Bins[1].MaximumDuration)
		}
		if history.Bins[0].Checks != 0 || !history.Bins[0].LastCheck.Equal(from.Add(-30*time.Second)) {
			t.Error("expected bin 0 to have no check and to reference the check that preceded the window")
		}
		if !history.Bins[5].LastCheck.Equal(from.Add(3*time.Minute + 30*time.Second)) {
			t.Error("expected bin 5 to reference the last check of bin 3")
		}
		// Bin 3 has a successful check followed by a failed check, so the condition results must be the failed one's
		if len(history.Bins[3].ConditionResults) != 1 || history.Bins[3].ConditionResults[0].Success {
			t.Error("expected bin 3 to have the condition results of its last check, which failed")
		}
		// Bin 5 has no check, so it must have the condition results of the check it was carried over from
		if len(history.Bins[5].ConditionResults) != 1 || history.Bins[5].ConditionResults[0].Success {
			t.Error("expected bin 5 to have the condition results of the check it was carried over from")
		}
	})
	t.Run("carry-over-limit", func(t *testing.T) {
		history := newHistory()
		history.Finalize(80 * time.Second)
		expected := []HistoryState{HistoryStateUp, HistoryStateUp, HistoryStateDown, HistoryStateDegraded, HistoryStateDown, HistoryStateUnknown}
		for i, bin := range history.Bins {
			if bin.State != expected[i] {
				t.Errorf("bin %d: expected state %s, got %s", i, expected[i], bin.State)
			}
		}
		if !history.Bins[5].LastCheck.IsZero() || history.Bins[5].ConditionResults != nil {
			t.Error("expected bin with unknown state to have no last check")
		}
	})
	t.Run("no-data", func(t *testing.T) {
		history := NewHistory(from, from.Add(time.Hour), 10)
		history.Finalize(0)
		for i, bin := range history.Bins {
			if bin.State != HistoryStateUnknown {
				t.Errorf("bin %d: expected state %s, got %s", i, HistoryStateUnknown, bin.State)
			}
		}
	})
}

func TestHistory_SetPreviousCheck(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	history := NewHistory(from, from.Add(time.Hour), 10)
	history.SetPreviousCheck(newTestResult(from.Add(-time.Minute), false, 0))
	history.SetPreviousCheck(newTestResult(from.Add(-2*time.Minute), true, 0)) // older, must be ignored
	history.SetPreviousCheck(newTestResult(from, true, 0))                     // not before the window, must be ignored
	history.Finalize(0)
	if history.Bins[0].State != HistoryStateDown {
		t.Errorf("expected first bin to be %s, got %s", HistoryStateDown, history.Bins[0].State)
	}
}

func TestHistory_LastFailedCheck(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	history := NewHistory(from, from.Add(3*time.Minute), 3)
	failedCheck := newTestResult(from.Add(10*time.Second), false, 0)
	failedCheck.Errors = []string{"error"}
	successfulCheck := newTestResult(from.Add(20*time.Second), true, 0)
	history.AddCheck(failedCheck)
	history.AddCheck(successfulCheck)
	lastCheckOfSecondBin := newTestResult(from.Add(time.Minute+10*time.Second), true, 0)
	history.AddCheck(lastCheckOfSecondBin)
	history.Finalize(0)
	// Bin 0: a failed check followed by a successful check
	bin := history.Bins[0]
	if bin.State != HistoryStateDegraded {
		t.Errorf("expected bin 0 to be %s, got %s", HistoryStateDegraded, bin.State)
	}
	if !bin.LastCheck.Equal(successfulCheck.Timestamp) || !bin.LastFailedCheck.Equal(failedCheck.Timestamp) {
		t.Errorf("expected bin 0 to have last check %s and last failed check %s, got %s and %s", successfulCheck.Timestamp, failedCheck.Timestamp, bin.LastCheck, bin.LastFailedCheck)
	}
	if len(bin.Errors) != 1 || len(bin.ConditionResults) != 1 || bin.ConditionResults[0].Success {
		t.Error("expected bin 0 to have the errors and condition results of the failed check")
	}
	// Bin 1: only a successful check
	bin = history.Bins[1]
	if !bin.LastFailedCheck.IsZero() || len(bin.Errors) != 0 || len(bin.ConditionResults) != 1 || !bin.ConditionResults[0].Success {
		t.Error("expected bin 1 to have no last failed check and the condition results of its last check")
	}
	// Bin 2: no check, carried over from the successful check of bin 1
	bin = history.Bins[2]
	if bin.State != HistoryStateUp || !bin.LastCheck.Equal(lastCheckOfSecondBin.Timestamp) || !bin.LastFailedCheck.IsZero() {
		t.Error("expected bin 2 to be carried over from the last check of bin 1, without last failed check")
	}
}
