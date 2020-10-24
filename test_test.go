package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/textproto"
	"net/url"
	"testing"
)

// mockServer is a basic HTTP server that sends pre-designated responses.
// When a client connects, mockServer reads until it encounters a blank line
// and then sends the next active response back to the client.
type mockServer struct {
	responses [][]string
	listener  *net.TCPListener
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
		for _, line := range ms.responses[i] {
			fmt.Fprint(clientSock, line)
		}
		clientSock.Close()
	}
}

func TestMakeHTTPRequest(t *testing.T) {
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
