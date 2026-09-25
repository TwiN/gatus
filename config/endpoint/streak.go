package endpoint

// NumberOfResultsInARow returns the trailing run of consecutive results whose Success
// matches the most recent result. results must be ordered oldest-first (newest last) —
// the same order returned by storage.Store.GetEndpointStatusByKey/GetSuiteStatusByKey —
// since this function walks the slice from the end without re-sorting by Result.Timestamp.
// Passing results in any other order will silently produce an incorrect streak.
// Exactly one return value is non-zero; both are zero when results is empty.
func NumberOfResultsInARow(results []*Result) (failuresInARow, successesInARow int) {
	for i := len(results) - 1; i >= 0; i-- {
		if results[i].Success {
			if failuresInARow > 0 {
				break
			}
			successesInARow++
		} else {
			if successesInARow > 0 {
				break
			}
			failuresInARow++
		}
	}
	return
}
