package endpoint

import (
	"time"
)

// NumberOfHistoryBins is the number of bins a History is split into
const NumberOfHistoryBins = 50

// HistoryState is the aggregated state of a HistoryBin
type HistoryState string

const (
	HistoryStateUp       HistoryState = "up"       // All checks within the bin were successful
	HistoryStateDegraded HistoryState = "degraded" // Some, but not all, checks within the bin were successful
	HistoryStateDown     HistoryState = "down"     // All checks within the bin failed
	HistoryStateUnknown  HistoryState = "unknown"  // No data is available for the bin
)

// History is the aggregated history of an endpoint's results over a time window,
// split into bins of equal width
type History struct {
	// Window is the duration covered by the history
	Window time.Duration `json:"window"`

	// Bins are the bins of the history, from oldest to newest
	Bins []*HistoryBin `json:"bins"`

	from, to time.Time

	// previousCheck is the last check that happened before the window,
	// so that its state can be carried over into the first bins of the window
	previousCheck *Result
}

// HistoryBin is the aggregation of all results that occurred within [Start, End)
type HistoryBin struct {
	// Start of the bin (inclusive)
	Start time.Time `json:"start"`

	// End of the bin (exclusive)
	End time.Time `json:"end"`

	// State of the bin
	State HistoryState `json:"state"`

	// Checks is the number of checks that occurred within the bin
	Checks int `json:"checks"`

	// SuccessfulChecks is the number of successful checks that occurred within the bin
	SuccessfulChecks int `json:"successfulChecks"`

	// AverageDuration is the average response time of the checks within the bin
	AverageDuration time.Duration `json:"averageDuration,omitempty"`

	// MinimumDuration is the lowest response time of the checks within the bin
	MinimumDuration time.Duration `json:"minimumDuration,omitempty"`

	// MaximumDuration is the highest response time of the checks within the bin
	MaximumDuration time.Duration `json:"maximumDuration,omitempty"`

	// LastCheck is the timestamp of the most recent check within the bin or, if the bin
	// has no check and its state was carried over from a previous check, the timestamp of said check
	LastCheck time.Time `json:"lastCheck,omitzero"`

	// LastCheckErrors are the errors of the check referenced by LastCheck
	LastCheckErrors []string `json:"lastCheckErrors,omitempty"`

	// LastCheckConditionResults are the condition results of the check referenced by LastCheck
	LastCheckConditionResults []*ConditionResult `json:"lastCheckConditionResults,omitempty"`

	totalDuration time.Duration
	lastResult    *Result
}

// NewHistory creates a History made of numberOfBins empty bins of equal width spanning [from, to)
func NewHistory(from, to time.Time, numberOfBins int) *History {
	if numberOfBins < 1 {
		numberOfBins = 1
	}
	history := &History{
		Window: to.Sub(from),
		Bins:   make([]*HistoryBin, numberOfBins),
		from:   from,
		to:     to,
	}
	binWidth := history.BinWidth()
	for i := range history.Bins {
		start := from.Add(time.Duration(float64(binWidth) * float64(i)))
		end := from.Add(time.Duration(float64(binWidth) * float64(i+1)))
		if i == numberOfBins-1 {
			end = to
		}
		history.Bins[i] = &HistoryBin{Start: start, End: end, State: HistoryStateUnknown}
	}
	return history
}

// NewHistoryWindow returns the time range [from, to) of a history covering the given window as of now.
//
// The end of the window is aligned on the next bin boundary so that bins remain stable across successive
// calls instead of shifting a little bit every time.
func NewHistoryWindow(now time.Time, window time.Duration, numberOfBins int) (from, to time.Time) {
	if numberOfBins < 1 {
		numberOfBins = 1
	}
	binWidth := int64(window) / int64(numberOfBins)
	if binWidth <= 0 {
		return now.Add(-window), now
	}
	to = time.Unix(0, (now.UnixNano()/binWidth+1)*binWidth)
	return to.Add(-window), to
}

// From returns the start of the history's time range (inclusive)
func (h *History) From() time.Time {
	return h.from
}

// To returns the end of the history's time range (exclusive)
func (h *History) To() time.Time {
	return h.to
}

// BinWidth returns the duration covered by each bin
func (h *History) BinWidth() time.Duration {
	return h.to.Sub(h.from) / time.Duration(len(h.Bins))
}

// BinIndex returns the index of the bin the timestamp falls in, or -1 if the timestamp is outside the history's range
func (h *History) BinIndex(timestamp time.Time) int {
	if timestamp.Before(h.from) || !timestamp.Before(h.to) {
		return -1
	}
	index := int(float64(timestamp.Sub(h.from)) / float64(h.to.Sub(h.from)) * float64(len(h.Bins)))
	return min(max(index, 0), len(h.Bins)-1)
}

// AddCheck adds a single check to the bin its timestamp falls in.
// Checks that happened before the history's range are used as the previous check.
func (h *History) AddCheck(result *Result) {
	if result == nil {
		return
	}
	if result.Timestamp.Before(h.from) {
		h.SetPreviousCheck(result)
		return
	}
	index := h.BinIndex(result.Timestamp)
	if index < 0 {
		return
	}
	h.AddAggregatedChecks(index, 1, boolToInt(result.Success), result.Duration, result.Duration, result.Duration, result)
}

// AddAggregatedChecks adds pre-aggregated checks to the bin at the given index.
//
// lastResult is the most recent of the aggregated checks. Only its Timestamp, Success, Errors and ConditionResults
// are used, and since it is used by reference, its Errors and ConditionResults may be populated until Finalize is called.
func (h *History) AddAggregatedChecks(index, checks, successfulChecks int, totalDuration, minimumDuration, maximumDuration time.Duration, lastResult *Result) {
	if index < 0 || index >= len(h.Bins) || checks <= 0 || lastResult == nil {
		return
	}
	bin := h.Bins[index]
	if bin.Checks == 0 || minimumDuration < bin.MinimumDuration {
		bin.MinimumDuration = minimumDuration
	}
	if bin.Checks == 0 || maximumDuration > bin.MaximumDuration {
		bin.MaximumDuration = maximumDuration
	}
	if bin.lastResult == nil || !lastResult.Timestamp.Before(bin.lastResult.Timestamp) {
		bin.lastResult = lastResult
	}
	bin.Checks += checks
	bin.SuccessfulChecks += successfulChecks
	bin.totalDuration += totalDuration
}

// SetPreviousCheck sets the check that happened before the history's range, if it is more recent than the current one.
//
// Like for AddAggregatedChecks, the result is used by reference.
func (h *History) SetPreviousCheck(result *Result) {
	if result == nil || !result.Timestamp.Before(h.from) || (h.previousCheck != nil && result.Timestamp.Before(h.previousCheck.Timestamp)) {
		return
	}
	h.previousCheck = result
}

// Finalize computes the state of every bin.
//
// Bins with checks are "up" if all checks were successful, "down" if all checks failed and "degraded" otherwise.
// Bins without any check inherit the state of the most recent check that preceded them, as long as said check
// is not older than maximumCarryOver (a maximumCarryOver of 0 or less means there is no limit). Otherwise, their
// state is "unknown".
//
// Every bin with a state other than "unknown" also references its last check (or the check its state was carried
// over from) through LastCheck, LastCheckErrors and LastCheckConditionResults.
func (h *History) Finalize(maximumCarryOver time.Duration) {
	lastResult := h.previousCheck
	for _, bin := range h.Bins {
		if bin.Checks > 0 {
			switch bin.SuccessfulChecks {
			case bin.Checks:
				bin.State = HistoryStateUp
			case 0:
				bin.State = HistoryStateDown
			default:
				bin.State = HistoryStateDegraded
			}
			bin.AverageDuration = bin.totalDuration / time.Duration(bin.Checks)
			lastResult = bin.lastResult
			bin.setLastCheck(lastResult)
			continue
		}
		bin.State = HistoryStateUnknown
		if lastResult == nil || (maximumCarryOver > 0 && bin.Start.Sub(lastResult.Timestamp) > maximumCarryOver) {
			continue
		}
		if lastResult.Success {
			bin.State = HistoryStateUp
		} else {
			bin.State = HistoryStateDown
		}
		bin.setLastCheck(lastResult)
	}
}

func (bin *HistoryBin) setLastCheck(result *Result) {
	bin.LastCheck = result.Timestamp
	bin.LastCheckErrors = result.Errors
	bin.LastCheckConditionResults = result.ConditionResults
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
