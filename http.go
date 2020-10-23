package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"jockey/counter"
	"net"
	"net/textproto"
	"net/url"
	"os"
	"strconv"
	"strings"
)


// parseFuzzyHttpUrl parses a user supplied URL.
// If the url contains a URL scheme other than http (i.e. https, wss, etc)
// an error is returned.
func parseFuzzyHttpUrl(urlRaw string) (*url.URL, error) {
	originalURL := urlRaw
	// Retry at most once to add scheme
	for i := 0; i < 2; i++ {
		parsed, err := url.Parse(urlRaw)
		if err != nil {
			return nil, fmt.Errorf("invalid URL %s\n", originalURL)
		}
		if parsed.Scheme == "" || parsed.Hostname() == "" {
			// url.Parse assumes a valid URL with a scheme
			// Retry with http if the user did not specify
			urlRaw = "http://" + urlRaw
			continue
		} else if parsed.Scheme != "http" {
			return nil, fmt.Errorf("invalid URL scheme: expected http, got %s", parsed.Scheme)
		}
		return parsed, nil
	}
	return nil, fmt.Errorf("invalid URL: %s\n", originalURL)
}

// dumpResponse makes an http request for a path at the given host and port
// and writes the response body to writer. On a successful request dumpResponse
// returns the HTTP status code and the number of bytes read (including headers),
// otherwise it returns an error.
func dumpResponse(writer io.Writer, host string, path string, port int,
	headers *map[string]string) (int, int, error) {

	conn, err := openRequest(host, path, port, headers)
	if err != nil {
		return 0, -1, err
	}
	defer conn.Close()

	// Count the number of bytes read by wrapping the connection in a counter.Reader
	counts := counter.NewReader(conn)
	reader := bufio.NewReaderSize(counts, os.Getpagesize()*16)
	tp := textproto.NewReader(reader)
	statusLine, err := tp.ReadLine()
	if err != nil {
		return counts.Count(), -1, err
	}

	// Parse the Status-Line
	statusFields := strings.Fields(statusLine)
	// Response code is the second field in a HTTP Status-Line
	status, err := strconv.Atoi(statusFields[1])
	if err != nil {
		// Bad Status-Line
		return counts.Count(), -1, errors.New(fmt.Sprintf("bad status line %s\n", statusLine))
	}
	// Skip over the rest of the headers without writing them to writer
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return counts.Count(), status, err
		}
		if line == "" {
			break
		}
	}

	// Write the response body to writer
	_, err = reader.WriteTo(writer)
	if err != nil && err != io.EOF {
		return counts.Count(), status, err
	}
	return counts.Count(), status, nil
}

// Send an HTTP request and return an open TCP connection
func openRequest(host string, path string, port int, headers *map[string]string) (*net.TCPConn, error) {
	// Based on the way I'm interpreting the assignment I'm specifically avoiding
	// the net.Dial function and manually looking up the host address

	reqHeaders := map[string]string{
		"Host":            fmt.Sprintf("%s:%d", host, port),
		"User-Agent":      "Mozilla/5.0",
		"Accept":          "*/*",
		"Accept-Encoding": "identity",
		"Connection":      "close",
	}
	// The caller is reasonable for providing reasonable headers if they override defaults
	if headers != nil {
		for k, v := range *headers {
			reqHeaders[k] = v
		}
	}
	if path == "" {
		path = "/"
	}

	addrs, _ := net.LookupHost(host)
	// TODO: This seems fragile -> maybe I can just use net.Dial since all the spec mentioned is
	// 	not using a library for HTTP
	for _, addr := range addrs {
		tcpAddr := &net.TCPAddr{IP: net.ParseIP(addr), Port: port}
		// TODO: Add a timeout here
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			continue
		}
		// Write HTTP request line and headers
		writer := bufio.NewWriter(conn)
		// Buffered writer will noop after first error so we only check on final Flush()
		_, _ = fmt.Fprintf(writer, "GET %s HTTP/1.1\r\n", path)
		for header, value := range reqHeaders {
			_, _ = fmt.Fprintf(writer, "%s: %s\r\n", header, value)
		}
		_, _ = fmt.Fprint(writer, "\r\n")
		err = writer.Flush()
		if err != nil {
			conn.Close()
			return nil, err
		}
		return conn, nil
	}
	// Exhausted all host address options without connecting
	return nil, errors.New("could not connect to host")
}
