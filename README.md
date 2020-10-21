## Jockey - A simple HTTP benchmark tool
Jockey is a simple tool for sending HTTP requests and profiling a web server.

## Build
Jockey is written in Go and tested with Go 14.4

To build, simply run `go build` from within the repository root directory.

## Usage
```
Usage: ./jockey -url <URL>
Options:
  -profile int
    	Make n requests to the target URL and print request statistics (default -1)
  -url string
    	(Required) The URL to send HTTP requests. Defaults to port 80 unless specified by URL:PORT

By default, Jockey sends a single HTTP request to the specified URL and dumps
the body of the HTTP response to stdout. Use the --profile option to send n
requests and generate profile report.
```

## Cloudflare
Written based on the requirements of Cloudflare's
[2020 systems engineering assessment](https://github.com/cloudflare-hiring/cloudflare-2020-systems-engineering-assignment).