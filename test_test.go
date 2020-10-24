package main

import (
	"net/url"
	"testing"
)

func TestParseFuzzyHttpUrl(t *testing.T) {
	// URL test case for parseFuzzyHttpUrl
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
		got, err := parseFuzzyHttpUrl(testCase.rawURL)
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
