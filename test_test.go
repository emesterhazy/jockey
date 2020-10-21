package main

import (
	"math/rand"
	"net/url"
	"testing"
	"time"
)

// Helper function to compare result returned by ParseURL
func doTestParseURL(t *testing.T, testCase *urlCase) {
	got, err := parseFuzzyURL(testCase.rawURL)
	if err != nil && !testCase.expectError {
		t.Errorf("Parsing %s returned error%s\n",
			err.Error(), testCase.rawURL)
		return
	} else if err != nil && testCase.expectError {
		return
	} else if err == nil && testCase.expectError {
		t.Errorf("Expected error parsing %s and got none\n", testCase.rawURL)
	} else if err != nil && got == nil {
		t.Errorf("parseFuzzyURL returned a nil pointer without setting error for url %s\n",
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

func TestPartition(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	mx := 10_000
	nums := make([]time.Duration, mx)
	for i := range nums {
		nums[i] = time.Duration(i + 1)
	}
	for i := 0; i < 1000; i++ {
		rand.Shuffle(len(nums), func(i, j int) {
			nums[i], nums[j] = nums[j], nums[i]
		})
		piv := rand.Intn(len(nums))
		piv = partition(nums, 0, len(nums)-1, piv)
		// No numbers below pivot are greater
		for j := 0; j < piv; j++ {
			if nums[j] > nums[piv] {
				t.Errorf("failed partition")
			}
		}
		// No numbers above pivot are smaller
		for j := piv + 1; j < len(nums); j++ {
			if nums[j] < nums[piv] {
				t.Errorf("failed partition")
			}
		}
	}
}

func TestQuickSelect(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	mx := 10_000
	nums := make([]time.Duration, mx)
	for i := range nums {
		nums[i] = time.Duration(i + 1)
	}

	for i := 0; i < 1000; i++ {
		rand.Shuffle(len(nums), func(i, j int) {
			nums[i], nums[j] = nums[j], nums[i]
		})
		// These type casts get a little hairy in Go
		exp := time.Duration(rand.Intn(mx) + 1)
		if val, _ := quickSelect(nums, int(exp)); val != exp {
			t.Errorf("wrong value expected %d got %d\n", exp, val)
		}
	}
}
