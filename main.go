package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

type profileFlag struct {
	set   bool
	value int
}

func (pf *profileFlag) Set(val string) error {
	i, err := strconv.Atoi(val)
	if err != nil {
		pf.value = 0
	} else {
		pf.value = i
	}
	pf.set = true
	return nil
}

func (pf *profileFlag) String() string {
	return strconv.Itoa(pf.value)
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s -url <URL>\nOptions:\n", os.Args[0])
	flag.PrintDefaults()
	msg := `
By default, Jockey sends a single HTTP request to the specified URL and dumps
the body of the HTTP response to stdout.

If the --profile <n> option is passed, Jockey sends n sequential requests and
generates a basic statistical report summarizing the outcome. Jockey considers
any HTTP status code >= 400 as an unsuccessful request and prints a count for
each unsuccessful error code it receives during the profile run. No status code
is printed for requests that fail due to broken network connections or invalid
HTTP responses.

On Unix based systems you can interrupt the profile at any point by sending
Jockey SIGINT, usually by pressing <Ctrl-C>. Jockey will attempt to quickly
complete its current request and exit after printing the statistics for any
completed requests.

Jockey currently only supports HTTP requests and does not follow redirects.
`
	fmt.Fprint(flag.CommandLine.Output(), msg)
}

func main() {
	flag.Usage = usage
	targetURL := flag.String(
		"url",
		"",
		"(Required) The URL to send HTTP Requests. Defaults to port 80 unless specified in the URL")
	var profileOpt profileFlag
	flag.Var(&profileOpt, "profile", "Make n requests to the target URL and print request statistics")
	flag.Parse()

	if *targetURL == "" {
		flag.Usage()
		os.Exit(1)
	}
	// Parse URL supplied by user
	parsed, err := ParseFuzzyHTTPUrl(*targetURL)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Make a single request to the url and dump the response to stdout
	if !profileOpt.set {
		_, _, err := MakeHTTPRequest(parsed, io.Writer(os.Stdout), nil, nil)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if profileOpt.value > 0 {
		// Run a profile on the url
		fmt.Printf("Running profile with %d repetitions...", profileOpt.value)
		results := DoProfile(profileOpt.value, parsed, nil)
		fmt.Printf("\n%s", results.String())
	} else {
		_, _ = fmt.Fprintln(os.Stderr, "-profile requires a positive number of repetitions")
		os.Exit(1)
	}
	os.Exit(0)
}
