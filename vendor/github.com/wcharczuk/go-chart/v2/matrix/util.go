package matrix

import (
	"math"
	"strconv"
)

func minInt(values ...int) int {
	min := math.MaxInt32

	for x := 0; x < len(values); x++ {
		if values[x] < min {
			min = values[x]
		}
	}
	return min
}

func maxInt(values ...int) int {
	max := math.MinInt32

	for x := 0; x < len(values); x++ {
		if values[x] > max {
			max = values[x]
		}
	}
	return max
}

func f64s(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func roundToEpsilon(value, epsilon float64) float64 {
	return math.Nextafter(value, value)
}
