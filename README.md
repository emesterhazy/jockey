## Jockey - A simple HTTP benchmark tool
Jockey is a simple tool for sending HTTP requests and profiling a web server.

## Build
Jockey is written in Go and tested on Linux and MacOS with Go 1.13, 1.14, and 1.15.

To build, simply run `go build` from within the repository root directory.

## Usage
```
Usage: ./jockey -url <URL>
Options:
  -profile value
    	Make n requests to the target URL and print request statistics
  -url string
    	The URL to send HTTP requests. (Required)
    	Defaults to port http and 80 unless specified in the URL

By default, Jockey sends a single HTTP request to the specified URL and dumps
the body of the HTTP response to stdout.

Jockey supports both HTTP and HTTPS, and does not follow redirects.

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
```

## Examples
### Test Cases
All tests were run from NYC over Verizon FIOS and Ethernet. Results appear
sorted by median time to complete requests from fastest to slowest.

![Jockey Test Results](https://www.evanmesterhazy.com/images/jockey_examples.png)

### Discussion of Results
In absolute terms, https://worker-demo.evanmesterhazy.com and
https://www.evanmesterhazy.com exhibited both lower mean and median response
times than any other site. Both of these sites are served by Cloudflare;
https://worker-demo.evanmesterhazy.com is a [Cloudflare Workers](https://workers.cloudflare.com/)
site, and https://www.evanmesterhazy.com is a static site sitting behind
Cloudflare's CDN. The fact that the median response for these two sites is so
similar (even though the first is twice as large) is an indication of the speed
of Cloudflare's workers platform.

It's worth noting that even though these sites outperformed all others, they are
both considerably smaller in terms of bytes transferred. https://www.google.com
performed third fastest, trailed closely by https://www.cloudflare.com which
transferred over twice as many bytes. All other sites tested were at least two
times slower than https://worker-demo.evanmesterhazy.com.

## Acknowledgement
Jockey was written based on the requirements of Cloudflare's
[2020 systems engineering assessment](https://github.com/cloudflare-hiring/cloudflare-2020-systems-engineering-assignment).

## Author
Evan Mesterhazy - contact at evan.mesterhazy AT gmail.com
