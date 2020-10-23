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
the body of the HTTP response to stdout. Use the --profile option to send n
Requests and generate profile report.
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
	flag.Var(&profileOpt, "profile", "Make n Requests to the target URL and print request statistics")
	flag.Parse()

	if *targetURL == "" {
		flag.Usage()
		os.Exit(1)
	}
	// Parse URL supplied by user
	parsed, err := parseFuzzyHttpUrl(*targetURL)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	port := 80 // Use port 80 by default
	if parsed.Port() != "" {
		port, _ = strconv.Atoi(parsed.Port())
	}
	// Use http for all requests instead of https
	if parsed.Scheme == "https" {
		parsed.Scheme = "http"
	}

	// Make a single request to the url and dump the response to stdout
	if !profileOpt.set {
		_, _, err := dumpHTTP(io.Writer(os.Stdout), parsed.Hostname(), parsed.Path, port, nil)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	} else if profileOpt.value > 0 {
		// Run a profile on the url
		results := DoProfile(profileOpt.value, parsed.Hostname(), parsed.Path, port, nil)
		fmt.Print(results.String())
		os.Exit(0)
	} else {
		_, _ = fmt.Fprintln(os.Stderr, "-profile requires a positive number of repetitions")
		os.Exit(1)
	}
}
