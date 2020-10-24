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
	"regexp"
	"strconv"
	"strings"
	"time"
)

func MakeHTTPRequest(requestURL *url.URL, writer io.Writer, headers *map[string]string,
	abort chan time.Duration) (int, int, error) {

	conn, err := openRequest(requestURL, headers)
	if err != nil {
		return 0, -1, err
	}
	return readResponse(conn, writer, abort)
}

// parseFuzzyHttpUrl parses a user supplied URL.
// If the url contains a URL scheme other than http (i.e. https, wss, etc)
// an error is returned.
func parseFuzzyHttpUrl(urlRaw string) (*url.URL, error) {
	originalURL := urlRaw

	// Jockey only supports the http scheme and returns an error if the user
	// specifies any other scheme. If no scheme is specified http is assumed.
	// Using regex here allows us to return a more informative error message.
	re := regexp.MustCompile("^([a-z]+)://")
	foundScheme := re.FindStringSubmatch(originalURL)
	if foundScheme == nil {
		urlRaw = "http://" + urlRaw
	} else if foundScheme[1] != "http" {
		return nil, fmt.Errorf("incompatible URL scheme: expected http, got %s", foundScheme[1])
	}

	parsedURL, err := url.Parse(urlRaw)
	if err != nil {
		return nil, err
	}
	if parsedURL.Port() == "" {
		// Assign default HTTP port
		parsedURL.Host = parsedURL.Host + ":80"
	}
	return parsedURL, nil
}

// readResponse reads an HTTP response from an established TCP connection and writes
// the response body to writer. The caller must send a valid HTTP request over conn
// before passing it to readResponse.
//
// Reading from a TCP connection blocks the Go routine executing readResponse
// until the server closes the connection. The caller can abort long reads at
// any time (or never) by passing a timeout value over the abort channel.
// If a timeout is received over abort, readResponse will close conn after the
// specified timeout. Closing the abort channel closes conn immediately.
//
// Returns the HTTP status code of the response and the number of bytes read
// from the response including headers.
func readResponse(conn net.Conn, writer io.Writer, abort chan time.Duration) (int, int, error) {
	defer conn.Close()
	// Close the socket to unblock read if the caller decides to abort the request
	if abort != nil {
		cleanupChan := make(chan struct{})
		defer close(cleanupChan)
		go func() {
			select {
			case gracePeriod, ok := <-abort:
				if ok {
					// Give the connection a small amount of time to finish up
					timeout := time.NewTimer(gracePeriod)
					<-timeout.C
				}
				conn.Close()
			case <-cleanupChan:
				// Avoid leaking the Go routine if no signal is received
			}
		}()
	}

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

// openRequest opens a TCP connection to the host specified in requestURL and sends
// a HTTP request corresponding to the request URI in requestURL using a set of
// default HTTP headers and any headers passed by the caller. Header values passed
// by the caller take precedence over defaults. By default the server is instructed
// to close the connection after sending its response.
//
// On success, openRequest returns a net.Conn representing the TCP connection.
// Returns an error if a connection cannot be established and if an error is returned
// while writing the HTTP request.
func openRequest(requestURL *url.URL, headers *map[string]string) (net.Conn, error) {
	reqHeaders := map[string]string{
		"Host":            requestURL.Host,
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
	conn, err := net.Dial("tcp", requestURL.Host)
	if err != nil {
		return nil, err
	}
	// Write HTTP request line and headers. Buffered writer will noop after the
	// first error so we only need to check err on the final Flush()
	writer := bufio.NewWriter(conn)
	_, _ = fmt.Fprintf(writer, "GET %s HTTP/1.1\r\n", requestURL.RequestURI())
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
