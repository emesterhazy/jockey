package main

import (
	"net/url"
	"testing"
)

// Helper function to compare result returned by ParseURL
func doTestParseURL(t *testing.T, testCase *urlCase) {
	got, err := fuzzyParseURL(testCase.rawURL)
	if err != nil && !testCase.expectError {
		t.Errorf("Parsing %s returned error%s\n",
			err.Error(), testCase.rawURL)
		return
	} else if err != nil && testCase.expectError {
		return
	} else if err == nil && testCase.expectError {
		t.Errorf("Expected error parsing %s and got none\n", testCase.rawURL)
	} else if err != nil && got == nil {
		t.Errorf("fuzzyParseURL returned a nil pointer without setting error for url %s\n",
			testCase.rawURL)
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

type urlCase struct {
	rawURL   string
	expected url.URL
	// We could define specific errors and check those if we wanted to be more robust
	expectError bool
}

func TestParse(t *testing.T) {
	cases := []urlCase{
		{"www.google.com", url.URL{Scheme: "http", Host: "www.google.com"}, false},
		{"www.google.com:80", url.URL{Scheme: "http", Host: "www.google.com:80"}, false},
		{"http://google.com", url.URL{Scheme: "http", Host: "google.com"}, false},
		{"google.com:80", url.URL{Scheme: "http", Host: "google.com:80"}, false},
		{"www.cloudflare.com:8000", url.URL{Scheme: "http", Host: "www.cloudflare.com:8000"}, false},
		{"wss://google.com", url.URL{}, true},
		{"http://www.cloudflare.com:80/index.html", url.URL{Scheme: "http", Host: "www.cloudflare.com:80"}, false},
		{"www.google.com:badport", url.URL{}, true},
	}
	for _, c := range cases {
		doTestParseURL(t, &c)
	}
}
