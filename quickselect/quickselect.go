package quickselect

import (
	"errors"
	"math/rand"
	"time"
)

// partition is the partitioning function in the quick select algorithm
// The slice times is partitioned such that all values smaller than
// times[pivotIndex] are at a lower index and all values greater than
// times[pivotIndex] are at a higher index in the slice.
func partition(times []time.Duration, left int, right int, pivotIndex int) int {
	pv := times[pivotIndex]
	times[pivotIndex], times[right] = times[right], times[pivotIndex]
	currentIx := left
	for i := left; i < right; i++ {
		if times[i] < pv {
			times[currentIx], times[i] = times[i], times[currentIx]
			currentIx++
		}
	}
	times[currentIx], times[right] = times[right], times[currentIx]
	return currentIx
}

// doQuickSelect implements the quickSelect algorithm
// https://en.wikipedia.org/wiki/Quickselect
func doQuickSelect(times []time.Duration, left int, right int, k int) time.Duration {
	if left == right {
		return times[left]
	}
	pivIndex := rand.Intn(right-left+1) + left
	pivIndex = partition(times, left, right, pivIndex)
	if k == pivIndex {
		return times[k]
	} else if k < pivIndex {
		return doQuickSelect(times, left, pivIndex-1, k)
	}
	// else
	return doQuickSelect(times, pivIndex+1, right, k)
}

// QuickSelect finds the kth largest item in slice times using the quick select
// algorithm where k is an index starting from 1.
// Quick select finds the kth largest item in O(n) time without sorting the
// slice and can be used to calculate the median value in a slice
// NOTE: QuickSelect changes the order of the values in times. The caller should
// pass a copy of the slice if this is undesirable.
func QuickSelect(times []time.Duration, k int) (time.Duration, error) {
	if k < 1 {
		return 0, errors.New("quickSelect: k less than 1")
	} else if len(times) == 0 {
		return 0, errors.New("quickSelect: empty slice")
	}
	return doQuickSelect(times, 0, len(times)-1, k-1), nil
}
