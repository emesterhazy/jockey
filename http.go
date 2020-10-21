package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
)

// Parse a user supplied URL string and accounting for missing http / https scheme
// by appending and retrying the parse
func parseFuzzyURL(rawURL string) (*url.URL, error) {
	originalURL := rawURL
	// Retry at most once to add scheme
	for i := 0; i < 2; i++ {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL %s\n", originalURL)
		}
		if parsed.Scheme == "" || (parsed.Scheme != "" && parsed.Hostname() == "") {
			// URL did not specify scheme do we default to http and retry parse
			// url.Parse is dumb and considers everything before the first : to be the scheme
			// which causes issues if no scheme is provided but :port is
			rawURL = "http://" + rawURL
			continue
		} else if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return nil, fmt.Errorf("invalid URL: %s\n", originalURL)
		}
		return parsed, nil
	}
	return nil, fmt.Errorf("invalid URL: %s\n", originalURL)
}

// Make an HTTP request and dump the response to [writer]
// Returns the HTTP response status
func dumpHTTP(writer io.Writer, host string, path string, port int,
	headers *map[string]string) (int, error) {

	conn, err := openRequest(host, path, port, headers)
	if err != nil {
		return -1, err
	}
	defer conn.Close()
	// TODO: keep track of the number of bytes read
	// New buffered reader with 16 page buffer
	reader := bufio.NewReaderSize(conn, 0x1000*16)
	tp := textproto.NewReader(reader)
	statusLine, err := tp.ReadLine()
	if err != nil {
		return -1, err
	}
	statusFields := strings.Fields(statusLine)
	status, err := strconv.Atoi(statusFields[1])
	if err != nil {
		// Bad status line
		return -1, errors.New(fmt.Sprintf("bad status line %s\n", statusLine))
	}
	// Skip headers
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return status, err
		}
		if line == "" {
			break
		}
	}

	_, err = reader.WriteTo(writer)
	if err != nil && err != io.EOF {
		return status, err
	}
	return status, nil

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
	if headers != nil {
		for k, v := range *headers {
			reqHeaders[k] = v
		}
	}
	if path == "" {
		path = "/"
	}

	addrs, _ := net.LookupHost(host)
	for _, addr := range addrs {
		tcpAddr := &net.TCPAddr{IP: net.ParseIP(addr), Port: port}
		// TODO: Add a timeout here
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			log.Print(err)
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
