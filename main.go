package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s -url <URL>\nOptions:\n", os.Args[0])
	flag.PrintDefaults()
	msg := `
By default, Jockey sends a single HTTP request to the specified URL and dumps
the body of the HTTP response to stdout. Use the --profile option to send n
requests and generate profile report.
`
	fmt.Fprint(flag.CommandLine.Output(), msg)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Usage = usage
	targetURL := flag.String(
		"url",
		"",
		"(Required) The URL to send HTTP requests. Defaults to port 80 unless specified by URL:PORT")
	profileOpt := flag.Int("profile", -1, "Make n requests to the target URL and print request statistics")
	flag.Parse()

	if *targetURL == "" {
		flag.Usage()
		os.Exit(1)
	}
	// Parse URL supplied by user
	parsed, err := parseFuzzyURL(*targetURL)
	if err != nil {
		log.Fatal(err)
	}
	port := 80 // Use port 80 by default
	if parsed.Port() != "" {
		port, _ = strconv.Atoi(parsed.Port())
	}
	// Use http for all requests instead of http
	if parsed.Scheme == "https" {
		parsed.Scheme = "http"
	}

	if *profileOpt < 0 {
		// Make a single request and dump the response to stdout
		_, _, err := dumpHTTP(io.Writer(os.Stdout), parsed.Hostname(), parsed.Path, port, nil)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	} else {
		results := DoProfile(*profileOpt, parsed.Hostname(), parsed.Path, port, nil)
		fmt.Print(results.String())
		os.Exit(0)
	}
}
