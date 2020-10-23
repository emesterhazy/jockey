package main

import (
	"fmt"
	"io/ioutil"
	"jockey/quickselect"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Status
// 3xx codes and below are not considered errors
const http_error_start = 400

// ProfileResults stores the results of the current profile run
type ProfileResults struct {
	Requests              uint
	FailedRequests        uint
	Fastest               time.Duration
	Slowest               time.Duration
	MeanTime              float64 // Float to minimize precision loss since we update on each request
	SmallestResponseBytes int
	LargestResponseBytes  int
	StatusCodeCounts      map[int]int
	// Median should be accessed through GetMedian since updating it is an O(n) operation
	requestTimes  []time.Duration
	medianTime    time.Duration
	medianCurrent bool // Avoid re-calculating if the median is up-to-date
}

// Init initializes a new ProfileResults struct
func (pr *ProfileResults) Init(numExpectedRequests int) {
	// Seed the random number generator for calculating the median later
	rand.Seed(time.Now().UnixNano())
	pr.StatusCodeCounts = make(map[int]int)
	pr.requestTimes = make([]time.Duration, 0, numExpectedRequests)
	pr.Fastest = math.MaxInt64
	pr.SmallestResponseBytes = math.MaxInt32
}

// String returns a formatted string representing the current results of the profile
func (pr *ProfileResults) String() string {
	const (
		minWidth = 0
		tabWidth = 0
		padding  = 2
		padChar  = ' '
		flags    = 0
	)
	var resultsBuilder strings.Builder
	writer := tabwriter.NewWriter(&resultsBuilder, minWidth, tabWidth, padding, padChar, flags)
	// Writes to tabwriter and string.Builder should not fail and there is not much
	// we can do if they do, so we just explicitly ignore the errors.
	percentSuccessful := float64(pr.Requests-pr.FailedRequests) / float64(pr.Requests) * 100
	_, _ = fmt.Fprintf(writer, "Requests:\t%15v\n", pr.Requests)
	_, _ = fmt.Fprintf(writer, "Successful Requests:\t%15.2f\t%%\n", percentSuccessful)
	_, _ = fmt.Fprintf(writer, "Fastest request:\t%15v\tms\n", pr.Fastest.Milliseconds())
	_, _ = fmt.Fprintf(writer, "Slowest request:\t%15v\tms\n", pr.Slowest.Milliseconds())
	_, _ = fmt.Fprintf(writer, "Mean time:\t%15v\tms\n", time.Duration(pr.MeanTime).Milliseconds())
	_, _ = fmt.Fprintf(writer, "Median time:\t%15v\tms\n", pr.GetMedian().Milliseconds())
	_, _ = fmt.Fprintf(writer, "Smallest Response:\t%15v\tbytes\n", pr.SmallestResponseBytes)
	_, _ = fmt.Fprintf(writer, "Largest Response:\t%15v\tbytes\n", pr.LargestResponseBytes)

	statusCodes := make([]int, 0, len(pr.StatusCodeCounts))
	for code := range pr.StatusCodeCounts {
		if code >= http_error_start {
			statusCodes = append(statusCodes, code)
		}
	}
	sort.IntSlice.Sort(statusCodes)
	if len(statusCodes) > 0 {
		_, _ = fmt.Fprintf(writer, "Error codes returned:\t\n")
		for _, code := range statusCodes {
			_, _ = fmt.Fprintf(writer, "%d:\t%15v\n", code, pr.StatusCodeCounts[code])
		}
	}
	_ = writer.Flush()
	return resultsBuilder.String()
}

// GetMedian gets the median response time from the current set of test results
// Quick select is used to determine the median since it runs in O(n) time and
// Jockey only calculates the median once per run. If Jockey needed to calculate
// the median more than once this function could be implemented using two priority
// queues which would require O(n log n) overall but would allow the median to be
// updated after each run in O(log n) time.
func (pr *ProfileResults) GetMedian() time.Duration {
	if pr.medianCurrent || len(pr.requestTimes) == 0 {
		return pr.medianTime
	}
	median, _ := quickselect.Median(pr.requestTimes)
	pr.medianCurrent = true
	return median
}

// UpdateStats updates the profile results to incorporate the results of a single test
// Use RecordFailedTransaction if the attempted transaction failed without a valid HTTP
// response.
func (pr *ProfileResults) UpdateStats(status int, requestTime time.Duration,
	bytesTransferred int) {
	pr.Requests++
	// The mean request time can be updated in O(1) time on each result
	// Add the request to pr.requestTimes first since we take the length in
	// order to calculate the mean using only requests with an associated
	// time (excluding requests that failed without a status code).
	pr.requestTimes = append(pr.requestTimes, requestTime)
	delta := float64(requestTime) - pr.MeanTime
	pr.MeanTime += delta / float64(len(pr.requestTimes))

	// Update Slowest / Fastest response
	if requestTime > pr.Slowest {
		pr.Slowest = requestTime
	}
	if requestTime < pr.Fastest {
		pr.Fastest = requestTime
	}
	// Update largest / smallest response
	if bytesTransferred > pr.LargestResponseBytes {
		pr.LargestResponseBytes = bytesTransferred
	}
	if bytesTransferred < pr.SmallestResponseBytes {
		pr.SmallestResponseBytes = bytesTransferred
	}
	// Need to store all times to calculate median...
	pr.medianCurrent = false

	if _, ok := pr.StatusCodeCounts[status]; ok {
		pr.StatusCodeCounts[status]++
	} else {
		pr.StatusCodeCounts[status] = 1
	}
	if status >= http_error_start {
		pr.FailedRequests++
	}
}

// RecordFailedTransaction records an attempted request that result in an error
// without receiving a valid HTTP response, such as a broken pipe, refused connection
// or malformed HTTP response.
func (pr *ProfileResults) RecordFailedTransaction() {
	pr.Requests++
	pr.FailedRequests++
}

// DoProfile sends HTTP GET requests for path to server host on the specified port
// and records statistics based on the requests. The number of requests sent is
// specified by the repetitions argument.
// Returns a ProfileResults struct with the results of the profile run.
func DoProfile(repetitions int, host string, path string, port int,
	headers *map[string]string) *ProfileResults {

	results := &ProfileResults{}
	results.Init(repetitions)

	// Set up signal handler to terminate early and print stats on sigint
	sigintChan := make(chan os.Signal, 1)
	signal.Notify(sigintChan, os.Interrupt)

	for i := 0; i < repetitions; i++ {
		start := time.Now()
		bytesRead, status, err := dumpResponse(ioutil.Discard, host, path, port, headers)
		if err != nil {
			results.RecordFailedTransaction()
			continue
		}
		stop := time.Now()
		elapsed := stop.Sub(start)
		results.UpdateStats(status, elapsed, bytesRead)
		select {
		// Break out of loop on sigint
		case <-sigintChan:
			return results
		default:
		}
	}
	return results
}
