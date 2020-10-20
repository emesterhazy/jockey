package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"net/url"
	"os"
	"strconv"
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

	parsed, err := fuzzyParseURL(*targetURL)
	if err != nil {
		log.Fatal(err)
	}

	port := 80
	if parsed.Port() != "" {
		port, _ = strconv.Atoi(parsed.Port())
	}
	conn, err := request(parsed.Hostname(), parsed.Path, port, nil)
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
		//fmt.Println(line)
		if line == "" {
			break
		}
	}
	buf := make([]byte, 4096*16)
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			conn.Close()
			log.Fatal(err)
		} else if err == io.EOF {
			err := conn.Close()
			os.Stdout.Write(buf[:n])
			if err != nil {
				log.Print("error closing connection")
				log.Print(err)
			}
			//fmt.Println("end of request response")
			break
		}
		os.Stdout.Write(buf[:n])
	}

	os.Exit(0)
}

// Parses a user supplied URL string and accounts for missing http / https scheme
// by appending and retrying the parse
func fuzzyParseURL(rawURL string) (*url.URL, error) {
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

func request(host string, path string, port int, headers *map[string]string) (*net.TCPConn, error) {
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
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			log.Print(err)
			continue
		}
		//reqLine := fmt.Sprintf("GET %s HTTP/1.1\r\n\r\n", path)
		//b := []byte(reqLine)
		fmt.Fprintf(conn, "GET %s HTTP/1.1\r\n", path)
		for header, value := range reqHeaders {
			fmt.Fprintf(conn, "%s: %s\r\n", header, value)
		}
		fmt.Fprint(conn, "\r\n")

		return conn, nil
	}
	return nil, errors.New("could not connect to host")
}
