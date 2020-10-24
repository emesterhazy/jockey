package main

import (
	"bufio"
	"crypto/tls"
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

const urlSchemeRegex = "^([a-zA-Z]+)://"

// See https://tools.ietf.org/html/rfc2616#page-40
// Note that this excludes the trailing \r\n since it is stripped prior to matching
const statusLineRegex = `^(?:HTTP|http)/\d\.\d (\d{3}) (?:[\x21-\x7E\x80-\xFF][\x20-\x7E\x80-\xFF]*)*$`

// MakeHTTPRequest opens a TCP connection to the host specified in requestURL and
// sends a single HTTP GET request corresponding to the request URI in requestURL
// using a set of default HTTP headers and any headers passed by the caller. Header
// values passed by the caller take precedence over defaults.
//
// The HTTP response body (omitting headers) is written to writer.
// Returns the HTTP status code and the number of bytes read.
//
// The caller can abort a request by passing an abort channel as an argument. The
// request will be aborted after an optional timeout if a duration is written to
// the abort channel or if the channel is closed.
func MakeHTTPRequest(requestURL *url.URL, writer io.Writer, headers *map[string]string,
	abort chan time.Duration) (status int, bytesRead int, err error) {

	// SendRequest closes conn
	//var conn net.Conn
	var tcpConn net.Conn
	var conn net.Conn
	tcpConn, err = net.Dial("tcp", requestURL.Host)
	if err != nil {
		return
	}

	// Negotiate TLS if required
	if requestURL.Scheme == "https" {
		c := tls.Client(tcpConn,
			&tls.Config{ServerName: requestURL.Hostname(), InsecureSkipVerify: true})
		conn = net.Conn(c)
	} else {
		conn = tcpConn
	}

	err = SendRequest(conn, requestURL, headers)
	if err != nil {
		return
	}
	return ReadResponse(conn, writer, abort)
}

// ParseFuzzyHTTPUrl parses a user supplied URL and attempts to use the http
// scheme and port 80 as defaults if the user does not provide a scheme or port.
// Returns an error on invalid URLs or if any scheme other than http is specified.
func ParseFuzzyHTTPUrl(urlRaw string) (*url.URL, error) {
	originalURL := urlRaw
	validSchemes := map[string]bool{"http": true, "https": true}

	// Jockey only supports the http scheme and returns an error if the user
	// specifies any other scheme. If no scheme is specified http is assumed.
	// Using regex here allows us to return a more informative error message.
	re := regexp.MustCompile(urlSchemeRegex)
	foundScheme := re.FindStringSubmatch(originalURL)
	if foundScheme == nil {
		urlRaw = "http://" + urlRaw
	} else if _, ok := validSchemes[strings.ToLower(foundScheme[1])]; !ok {
		return nil, fmt.Errorf("incompatible URL scheme: expected http or https, got %s",
			strings.ToLower(foundScheme[1]))
	}

	parsedURL, err := url.Parse(urlRaw)
	if err != nil {
		return nil, err
	}
	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)
	if parsedURL.Port() == "" {
		// Assign defaults
		if parsedURL.Scheme == "http" {
			parsedURL.Host = parsedURL.Host + ":80"
		} else if parsedURL.Scheme == "https" {
			parsedURL.Host = parsedURL.Host + ":443"
		} else {
			// This is unreachable
			return nil, fmt.Errorf("incompatible URL scheme: expected http, got %s",
				parsedURL.Scheme)
		}
	}
	return parsedURL, nil
}

// ReadResponse reads an HTTP response from an established TCP connection and writes
// the response body to writer. The caller must send a valid HTTP request over conn
// before passing it to ReadResponse.
//
// Reading from a TCP connection blocks the Go routine executing ReadResponse
// until the server closes the connection. The caller can abort long reads at
// any time (or never) by passing a timeout value over the abort channel.
// If a timeout is received over abort, ReadResponse will close conn after the
// specified timeout. Closing the abort channel closes conn immediately.
//
// Returns the HTTP status code of the response and the number of bytes read
// from the response including headers.
func ReadResponse(conn net.Conn, writer io.Writer, abort chan time.Duration) (
	status int, bytesRead int, retErr error) {

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
	defer func() { bytesRead = counts.Count() }()

	reader := bufio.NewReaderSize(counts, os.Getpagesize()*16)
	tp := textproto.NewReader(reader)
	// Parse the Status-Line; response code is the second field
	// See https://www.w3.org/Protocols/rfc2616/rfc2616-sec6.html
	statusLineRegex := regexp.MustCompile(statusLineRegex)
	// Continue waiting for final status if interim status 100 is received
	// See: https://tools.ietf.org/html/rfc2616#section-10.1.1
	for {
		statusLine, err := tp.ReadLine()
		if err != nil {
			retErr = err
			return
		}
		slMatch := statusLineRegex.FindStringSubmatch(statusLine)
		if slMatch == nil {
			retErr = errors.New(fmt.Sprintf("bad status line: %s\n", statusLine))
			return
		}
		status, _ = strconv.Atoi(slMatch[1])
		if status != 100 {
			break
		}
	}
	// Skip over the headers without writing them to writer
	for {
		line, err := tp.ReadLine()
		if err != nil {
			retErr = err
			return
		}
		if line == "" {
			break
		}
	}

	// Write the response body to writer
	_, err := reader.WriteTo(writer)
	if err != nil && err != io.EOF {
		retErr = err
		return
	}
	return
}

// SendRequest sends a HTTP GET request corresponding to the request URI in requestURL
// to conn using a set of default HTTP headers and any headers passed by the caller.
// Header values passed by the caller take precedence over defaults. By default the
// server is instructed to close the connection after sending its response.
func SendRequest(conn net.Conn, requestURL *url.URL, headers *map[string]string) error {
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
	// Write HTTP request line and headers. Buffered writer will noop after the
	// first error so we only need to check err on the final Flush()
	writer := bufio.NewWriter(conn)
	_, _ = fmt.Fprintf(writer, "GET %s HTTP/1.1\r\n", requestURL.RequestURI())
	for header, value := range reqHeaders {
		_, _ = fmt.Fprintf(writer, "%s: %s\r\n", header, value)
	}
	_, _ = fmt.Fprint(writer, "\r\n")
	err := writer.Flush()
	if err != nil {
		conn.Close()
		return err
	}
	return nil
}
