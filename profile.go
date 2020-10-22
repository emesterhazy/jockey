package main

import (
	"fmt"
	"io/ioutil"
	"jockey/quickselect"
	"math"
	"strings"
	"text/tabwriter"
	"time"
)

type profileResults struct {
	requests              uint
	fastest               time.Duration
	slowest               time.Duration
	meanTime              float64       // Float to minimize precision loss since we update on each request
	medianTime            time.Duration // Avoid re-calculating if the median is up-to-date
	medianCurrent         bool
	smallestResponseBytes int
	largestResponseBytes  int
	statusCodeCounts      map[int]int
	requestTimes          []time.Duration
}

func (pr *profileResults) Init(numExpectedRequests int) {
	pr.statusCodeCounts = make(map[int]int)
	pr.requestTimes = make([]time.Duration, 0, numExpectedRequests)
	pr.fastest = math.MaxInt64
	pr.smallestResponseBytes = math.MaxInt32
}

func (pr *profileResults) String() string {
	var str strings.Builder
	writer := tabwriter.NewWriter(&str, 0, 0, 4, ' ', 0)
	fmt.Fprintf(writer, "Requests:\t%8v\n", pr.requests)
	fmt.Fprintf(writer, "Fastest request:\t%8v ms\n", pr.fastest.Milliseconds())
	fmt.Fprintf(writer, "Slowest request:\t%8v ms\n", pr.slowest.Milliseconds())
	fmt.Fprintf(writer, "Mean time:\t%8v ms\n", time.Duration(pr.meanTime).Milliseconds())
	fmt.Fprintf(writer, "Median time:\t%8v ms\n", pr.GetMedian().Milliseconds())
	fmt.Fprintf(writer, "Smallest Response:\t%8v bytes\n", pr.smallestResponseBytes)
	fmt.Fprintf(writer, "Largest Response:\t%8v bytes\n", pr.largestResponseBytes)
	writer.Flush()
	return str.String()
}

func (pr *profileResults) GetMedian() time.Duration {
	times := pr.requestTimes
	if pr.medianCurrent || len(pr.requestTimes) == 0 {
		return pr.medianTime
	}
	mid := len(pr.requestTimes) / 2
	a, _ := quickselect.QuickSelect(times, mid)
	if pr.requests%2 == 0 {
		b, _ := quickselect.QuickSelect(times, mid+1)
		pr.medianTime = (a + b) / 2 // We might lose a fraction of a nanosecond but that's ok
	} else {
		pr.medianTime = a
	}
	pr.medianCurrent = true
	return pr.medianTime
}

// Update the statistics to incorporate a new request
func (pr *profileResults) UpdateStats(status int, requestTime time.Duration,
	bytesTransferred int) {
	pr.requests++
	// Online algorithm to update mean
	// TODO: Should I not bother with this and just use the values we're
	//  storing for the median to calculate it in one shot?
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

func DoProfile(repetitions int, host string, path string, port int,
	headers *map[string]string) *profileResults {
	results := &profileResults{}
	results.Init(repetitions)

	for i := 0; i < repetitions; i++ {
		start := time.Now()
		// TODO: dumpHTTP needs to return the number of bytes read
		bytesRead, status, err := dumpHTTP(ioutil.Discard, host, path, port, headers)
		if err != nil {
			status = 500
		}
		stop := time.Now()
		elapsed := stop.Sub(start)
		results.UpdateStats(status, elapsed, bytesRead)
	}

	return results
}
