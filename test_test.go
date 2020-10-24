package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/textproto"
	"net/url"
	"reflect"
	"testing"
	"time"
)

// mockServer is a basic HTTP server that sends pre-designated responses.
// When a client connects, mockServer reads until it encounters a blank line
// and then sends the next active response back to the client.
type mockServer struct {
	responses [][]string
	listener  *net.TCPListener
	// Optional delay between accepting a connection and responding
	delay time.Duration
}

// responseLengths calculates the length of the pre-designated responses.
// This is useful for verifying that the number of bytes read by MakeHTTPRequest
// is correct.
func (ms *mockServer) responseLengths() []int {
	lengths := make([]int, len(ms.responses))
	for i, response := range ms.responses {
		for _, line := range response {
			lengths[i] += len(line)
		}
	}
	return lengths
}

// start launches the mock server. The caller should run this function as a
// Go routine to avoid blocking. The caller should close the listener when it's
// done with the mockServer to avoid leaking its Go routine.
func (ms *mockServer) start(t *testing.T) {
	if ms.listener == nil || ms.responses == nil {
		t.Fatal("uninitialized mock server")
	}
	for i := 0; ; i = (i + 1) % len(ms.responses) {
		clientSock, err := ms.listener.Accept()
		if err != nil {
			return
		}
		textReader := textproto.NewReader(bufio.NewReader(clientSock))
		for {
			line, _ := textReader.ReadLine()
			if line == "" {
				break
			}
		}
		if ms.delay > 0 {
			timeout := time.NewTimer(ms.delay)
			<-timeout.C
		}
		for _, line := range ms.responses[i] {
			fmt.Fprint(clientSock, line)
		}
		clientSock.Close()
	}
}

func TestMakeHTTPRequestBasic(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal("error listening on localhost")
	}
	defer listener.Close()
	serverResponse := [][]string{{"HTTP/1.1 200 OK\r\n", "\r\n"}}
	ms := &mockServer{listener: listener.(*net.TCPListener), responses: serverResponse}
	go ms.start(t)

	parsedURL, err := url.Parse("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal("error parsing mock server url")
	}
	status, bytesRead, err := MakeHTTPRequest(parsedURL, ioutil.Discard, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if status != 200 {
		t.Errorf("expected HTTP status 200, got %d\n", status)
	}
	expectedLen := ms.responseLengths()[0]
	if bytesRead != expectedLen {
		t.Errorf("expected %d bytes read, got %d\n", expectedLen, bytesRead)
	}
}

// getRandomStatusLines is a helper function to return a random
// selection of valid HTTP status lines
// Returns an array of the complete status lines and a second array of
// integers representing the status code corresponding to each status line
func getRandomStatusLines(n int) ([]string, []int) {
	// Non-exhaustive list of HTTP status codes
	statusCodes := map[int]string{
		// Note that 100 Continue is a special case since it instructs the client
		// to discard it and continue with its request. As such, Jockey continues
		// reading the server's response and does not include the interim 100 in
		// its final output
		200: "OK",
		201: "Created",
		202: "Accepted",
		300: "Multiple Choice",
		301: "Moved Permanently",
		302: "Found",
		303: "See Other",
		400: "Bad Request",
		401: "Unauthorized",
		403: "Forbidden",
		404: "Not Found",
		408: "Request Timeout",
		418: "I'm a teapot",
		420: "Enhance Your Calm", // Twitter (unofficial)
		500: "Internal Server Error",
		504: "Gateway Time-out",
	}
	// Add all of the status codes to an array so we can randomly select them
	// to build the response lines
	options := make([]int, len(statusCodes))
	i := 0
	for code, _ := range statusCodes {
		options[i] = code
		i++
	}
	responseLines := make([]string, n)
	responseCodes := make([]int, n)
	for i = 0; i < n; i++ {
		ix := rand.Intn(len(statusCodes))
		code := options[ix]
		words, _ := statusCodes[code]
		responseLines[i] = fmt.Sprintf("HTTP/1.1 %d %s\r\n", code, words)
		responseCodes[i] = code
	}
	return responseLines, responseCodes
}

func TestResponseCodes(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal("error listening on localhost")
	}
	defer listener.Close()
	minResp := 100
	maxResp := 1000
	numberOfResponses := rand.Intn(maxResp-minResp+1) + minResp
	statusLines, statusCodes := getRandomStatusLines(numberOfResponses)
	// Construct complete responses for each status line
	serverResponses := make([][]string, len(statusLines))
	for i := range serverResponses {
		serverResponses[i] = []string{statusLines[i], "\r\n"}
	}
	// Calculate expected status line counts
	statusCodeCounts := make(map[int]int)
	for _, code := range statusCodes {
		if _, ok := statusCodeCounts[code]; ok {
			statusCodeCounts[code]++
		} else {
			statusCodeCounts[code] = 1
		}
	}

	ms := &mockServer{listener: listener.(*net.TCPListener), responses: serverResponses}
	go ms.start(t)

	parsedURL, err := url.Parse("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal("error parsing mock server url")
	}
	responses := DoProfile(numberOfResponses, parsedURL, nil)
	if !reflect.DeepEqual(responses.StatusCodeCounts, statusCodeCounts) {
		t.Errorf("mismatch in expected status codes:\ngot:      %v\nexpected: %v\n",
			responses.StatusCodeCounts, statusCodeCounts)
	}
}

// Test the ability to abort a blocked / long running request
func TestAbortHTTPRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal("error listening on localhost")
	}
	serverResponse := [][]string{{"HTTP/1.1 200 OK\r\n", "\r\n"}}
	serverDelay := time.Millisecond * 500
	ms := &mockServer{listener: listener.(*net.TCPListener), responses: serverResponse, delay: serverDelay}

	parsedURL, err := url.Parse("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal("error parsing mock server url")
	}
	// Set up a timeout to abort the connection
	abort := make(chan time.Duration)
	go ms.start(t)
	go func() {
		timeout := time.NewTimer(serverDelay / 2)
		<-timeout.C
		close(abort)
	}()
	_, _, err = MakeHTTPRequest(parsedURL, ioutil.Discard, nil, abort)
	// Expect an error here
	if err == nil {
		t.Errorf("connection did not abort")
	}
}

func TestBadResponseLines(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal("error listening on localhost")
	}
	defer listener.Close()
	serverResponse := [][]string{
		{"HTTP\r\n", "\r\n"},
		{"Jockey go fast\r\n", "\r\n"},
		{"HTTP/1.1 OK 200"},
		{"NOTHTTP/1.1 200 OK"},
	}
	ms := &mockServer{listener: listener.(*net.TCPListener), responses: serverResponse}
	go ms.start(t)

	parsedURL, err := url.Parse("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal("error parsing mock server url")
	}
	reps := len(serverResponse)
	results := DoProfile(reps, parsedURL, nil)
	if results.Requests != reps {
		t.Errorf("expected %d requests from profile got %d\n", reps, results.Requests)
	}
	if results.FailedRequests != reps {
		t.Errorf("expected %d failed requests from profile got %d\n", reps, results.FailedRequests)
	}

}

func TestParseFuzzyHttpUrl(t *testing.T) {
	// URL test case for ParseFuzzyHTTPUrl
	type urlCase struct {
		rawURL      string
		expected    url.URL
		expectError bool
	}
	cases := []urlCase{
		{"www.google.com", url.URL{Scheme: "http", Host: "www.google.com:80"}, false},
		{"www.google.com:80", url.URL{Scheme: "http", Host: "www.google.com:80"}, false},
		{"http://google.com", url.URL{Scheme: "http", Host: "google.com:80"}, false},
		{"google.com:80", url.URL{Scheme: "http", Host: "google.com:80"}, false},
		{"www.cloudflare.com:8000", url.URL{Scheme: "http", Host: "www.cloudflare.com:8000"}, false},
		{"http://www.cloudflare.com:80/index.html", url.URL{Scheme: "http", Host: "www.cloudflare.com:80"}, false},
		{"127.0.0.1:80", url.URL{Scheme: "http", Host: "127.0.0.1:80"}, false},
		{"127.0.0.1", url.URL{Scheme: "http", Host: "127.0.0.1:80"}, false},
		{"http://127.0.0.1:80", url.URL{Scheme: "http", Host: "127.0.0.1:80"}, false},
		{"[::1]:80", url.URL{Scheme: "http", Host: "[::1]:80"}, false},
		// Test cases that expect an error to be raised
		{"wss://www.google.com", url.URL{}, true},
		{"wss://google.com", url.URL{}, true},
		{"http://www.google.com:http", url.URL{}, true},
		{"www.google.com:badport", url.URL{}, true},
		{"https://www.google.com", url.URL{}, true},
	}
	for _, testCase := range cases {
		got, err := ParseFuzzyHTTPUrl(testCase.rawURL)
		if err != nil {
			if testCase.expectError {
				continue
			} else {
				t.Errorf("Parsing %s returned error%s\n", err.Error(), testCase.rawURL)
				continue
			}
		}
		if testCase.expectError {
			t.Errorf("Expected error parsing %s and got none\n", testCase.rawURL)
			continue
		}

		if got.Scheme != testCase.expected.Scheme {
			t.Errorf("Error parsing %s: got scheme %s expected %s\n",
				testCase.rawURL, got.Scheme, testCase.expected.Scheme)
		}
		if got.Host != testCase.expected.Host {
			t.Errorf("Error parsing %s: got host %s expected %s\n",
				testCase.rawURL, got.Host, testCase.expected.Host)
		}
	}
}
