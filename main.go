package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"net/url"
	"os"
)

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	targetURL := flag.String(
		"url",
		"",
		"(Required) The URL to send requests. Defaults to https unless http is specified")
	flag.Parse()

	if *targetURL == "" {
		flag.Usage()
		os.Exit(1)
	}

	parsed, err := parseURL(*targetURL)
	if err != nil {
		log.Fatal(err)
	}

	reqURL := parsed.Hostname() + "/" + parsed.Path
	conn, err := request(reqURL, 80)
	if err != nil {
		log.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)

	for {
		line, err := tp.ReadLine()
		if err != nil {
			break
		}
		fmt.Println(line)
	}
	conn.Close()

	os.Exit(0)
}

// Parses a user supplied URL string and accounts for missing http / https scheme
// by appending and retrying the parse
func parseURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %s\n", rawURL)
	}
	if parsed.Scheme != "" && !(parsed.Scheme == "https" || parsed.Scheme == "http") {
		// User did not supply scheme and did supply port so scheme got set to hostname
		if parsed.Hostname() != "" {
			return nil, fmt.Errorf("invalid URL scheme %s in %s\n", parsed.Scheme, rawURL)
		}
		parsed, err = url.Parse("http://" + rawURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %s\n", rawURL)
		}
	}
	return parsed, nil
}

func request(rawURL string, port int) (*net.TCPConn, error) {
	// Based on the way I'm interpreting the assignment I'm specifically avoiding the net.Dial function

	addrs, _ := net.LookupHost(rawURL)
	for _, addr := range addrs {
		tcpAddr := &net.TCPAddr{IP: net.ParseIP(addr), Port: port}
		fmt.Printf("%v\n", addr)
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			continue
		}
		return conn, nil
	}
	return nil, errors.New("could not connect to host")
}
