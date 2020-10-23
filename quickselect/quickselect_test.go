package quickselect

import (
	"math/rand"
	"testing"
	"time"
)

// Test the partition function in quickSelect
func TestPartition(t *testing.T) {
	// Generate mx random numbers to test with
	mx := 10_000
	rand.Seed(time.Now().UnixNano())
	nums := make([]time.Duration, mx)
	for i := range nums {
		nums[i] = time.Duration(i + 1)
	}

	// Shuffle and run 1000 test iterations and check that the numbers are
	// partitioned correctly
	for i := 0; i < 1000; i++ {
		rand.Shuffle(len(nums), func(i, j int) {
			nums[i], nums[j] = nums[j], nums[i]
		})
		pivotIndex := rand.Intn(len(nums))
		pivotIndex = partition(nums, 0, len(nums)-1, pivotIndex)
		// No numbers below pivot are greater
		for j := 0; j < pivotIndex; j++ {
			if nums[j] > nums[pivotIndex] {
				t.Errorf("failed partition: value larger than pivot below the pivot")
			}
		}
		// No numbers above pivot are smaller
		for j := pivotIndex + 1; j < len(nums); j++ {
			if nums[j] < nums[pivotIndex] {
				t.Errorf("failed partition: value smaller than pivot above the pivot")
			}
		}
	}
}

// Test the complete quick select algorithm
func TestQuickSelect(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	mx := 10_000
	nums := make([]time.Duration, mx)
	// Generate an array of sequential numbers so that the kth selection
	// is equal to k
	for i := range nums {
		nums[i] = time.Duration(i + 1)
	}

	// Shuffle and run 1000 iterations
	for i := 0; i < 1000; i++ {
		rand.Shuffle(len(nums), func(i, j int) {
			nums[i], nums[j] = nums[j], nums[i]
		})
		// Cast to time.Duration since the algorithm is typed to select on time.Durations
		exp := time.Duration(rand.Intn(mx) + 1)
		if val, _ := QuickSelect(nums, int(exp)); val != exp {
			t.Errorf("wrong value expected %d got %d\n", exp, val)
		}
	}
}

func TestMedianEven(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	numbers := []time.Duration{1, 10, 20, 30}
	if median, _ := Median(numbers); median != 15 {
		t.Errorf("median: expected %d got %d\n", 15, median)
	}
}

func TestMedianOdd(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	numbers := []time.Duration{1, 10, 20}
	if median, _ := Median(numbers); median != 10 {
		t.Errorf("median: expected %d got %d\n", 10, median)
	}
}
