package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"
	"text/tabwriter"
	"time"
)

type profileResults struct {
	requests              uint
	fastest               time.Duration
	slowest               time.Duration
	meanTime              float64
	medianTime            time.Duration // Avoid re-calculating if the median is up-to-date
	medianCurrent         bool
	smallestResponseBytes uint
	largestResponseBytes  uint
	statusCodeCounts      map[int]int
	requestTimes          []time.Duration
}

func (pr *profileResults) init(numExpectedRequests int) {
	pr.statusCodeCounts = make(map[int]int)
	pr.requestTimes = make([]time.Duration, 0, numExpectedRequests)
}

func (pr *profileResults) String() string {
	var str strings.Builder
	writer := tabwriter.NewWriter(&str, 0, 0, 4, ' ', 0)
	fmt.Fprintf(writer, "Requests:\t%v\n", pr.requests)
	fmt.Fprintf(writer, "Fastest request:\t%v ms\n", pr.fastest*time.Millisecond)
	fmt.Fprintf(writer, "Slowest request:\t%v ms\n", pr.slowest*time.Millisecond)
	fmt.Fprintf(writer, "Mean time:\t%v ms\n", time.Duration(pr.meanTime)*time.Millisecond)
	fmt.Fprintf(writer, "Median time:\t%v ms\n", pr.getMedian()*time.Millisecond)
	fmt.Fprintf(writer, "Smallest Response:\t%v bytes\n", pr.smallestResponseBytes)
	fmt.Fprintf(writer, "Largest Response:\t%v bytes\n", pr.largestResponseBytes)
	writer.Flush()
	return str.String()
}

func (pr *profileResults) getMedian() time.Duration {
	times := pr.requestTimes
	if pr.medianCurrent || len(pr.requestTimes) == 0 {
		return pr.medianTime
	}
	mid := len(pr.requestTimes) / 2
	a, _ := quickSelect(times, mid)
	if pr.requests%2 == 0 {
		b, _ := quickSelect(times, mid+1)
		pr.medianTime = (a + b) / 2 // We might lose a fraction of a nanosecond but that's ok
	} else {
		pr.medianTime = a
	}
	pr.medianCurrent = true
	return pr.medianTime
}

// Update the statistics to incorporate a new request
func (pr *profileResults) updateStats(status int, requestTime time.Duration,
	bytesTransferred uint) {
	pr.requests++
	// Online algorithm to update mean
	delta := float64(requestTime) - pr.meanTime
	pr.meanTime += delta / float64(pr.requests)
	// Update slowest / fastest response
	if requestTime > pr.slowest {
		pr.slowest = requestTime
	} else if requestTime < pr.fastest {
		pr.fastest = requestTime
	}
	// Update largest / smallest response
	if bytesTransferred > pr.largestResponseBytes {
		pr.largestResponseBytes = bytesTransferred
	} else if bytesTransferred < pr.smallestResponseBytes {
		pr.smallestResponseBytes = bytesTransferred
	}
	// Need to store all times to calculate median...
	pr.requestTimes = append(pr.requestTimes, requestTime)
	pr.medianCurrent = false

	if _, ok := pr.statusCodeCounts[status]; ok {
		pr.statusCodeCounts[status]++
	} else {
		pr.statusCodeCounts[status] = 1
	}
}

func doProfile(repetitions int, host string, path string, port int,
	headers *map[string]string) *profileResults {
	results := &profileResults{}
	results.init(repetitions)

	for i := 0; i < repetitions; i++ {
		start := time.Now()
		status, err := dumpHTTP(ioutil.Discard, host, path, port, headers)
		if err != nil {
			status = 500
		}
		stop := time.Now()
		elapsed := stop.Sub(start)
		results.updateStats(status, elapsed, 0)
	}

	return results
}

// Partition function for doQuickSelect
func partition(l []time.Duration, left int, right int, pivIndex int) int {
	pv := l[pivIndex]
	l[pivIndex], l[right] = l[right], l[pivIndex]
	currentIx := left
	for i := left; i < right; i++ {
		if l[i] < pv {
			l[currentIx], l[i] = l[i], l[currentIx]
			currentIx++
		}
	}
	l[currentIx], l[right] = l[right], l[currentIx]
	return currentIx
}

// The implements the quickSelect algorithm
// https://en.wikipedia.org/wiki/Quickselect
func doQuickSelect(l []time.Duration, left int, right int, k int) time.Duration {
	if left == right {
		return l[left]
	}
	pivIndex := rand.Intn(right-left+1) + left
	pivIndex = partition(l, left, right, pivIndex)
	if k == pivIndex {
		return l[k]
	} else if k < pivIndex {
		return doQuickSelect(l, left, pivIndex-1, k)
	}
	// else
	return doQuickSelect(l, pivIndex+1, right, k)
}

// Find the kth largest item in slice l with k indexed starting from 1
// quickSelect modifies the slice in place; make a copy if you want to avoid that
func quickSelect(l []time.Duration, k int) (time.Duration, error) {
	if k < 1 {
		return 0, errors.New("quickSelect: invalid k")
	} else if len(l) == 0 {
		return 0, errors.New("quickSelect: 0 length slice")
	}
	return doQuickSelect(l, 0, len(l)-1, k-1), nil
}
