# SiteMapper

![Build & Test](https://github.com/dinofizz/sitemapper/actions/workflows/ci.yml/badge.svg) [![codecov](https://codecov.io/gh/dinofizz/sitemapper/branch/main/graph/badge.svg?token=YOPGOKOEJB)](https://codecov.io/gh/dinofizz/sitemapper)

SiteMapper is a site mapping tool which provides a JSON output detailing each page visited, each with a list of links that are related to the root URL. The depth to which SiteMapper will explore a site is configurable, as well as the mode of operation: "synchronous", "concurrent" and "concurrent limited".

This README details the "standalone" CLI tool implementation. This repo also contains code related to running SiteMapper as a Kubernetes-based distributed crawler. See [README-k8s.md](./README-k8s.md),

Example:

```shell
2022/01/03 12:20:53 Using mode: concurrent
2022/01/03 12:20:53 Crawling https://google.com with depth 1
2022/01/03 12:20:53 visiting URL https://google.com at depth 0 with parent https://google.com
2022/01/03 12:20:53 Elapsed milliseconds:  312
[
  {
    "URL": "https://google.com",
    "Links": [
      "https://accounts.google.com/ServiceLogin",
      "https://drive.google.com/",
      "https://mail.google.com/mail/",
      "https://news.google.com/",
      "https://play.google.com/",
      "https://www.google.com/advanced_search",
      "https://www.google.com/intl/en/about.html",
      "https://www.google.com/intl/en/ads/",
      "https://www.google.com/intl/en/policies/privacy/",
      "https://www.google.com/intl/en/policies/terms/",
      "https://www.google.com/preferences",
      "https://www.google.com/services/",
      "https://www.google.com/setprefdomain"
    ]
  }
]
```
*Note: the above example pipes the output of SiteMapper into an additional JSON formatting tool, "[jq](https://stedolan.github.io/jq/)".*

**This tool is _not_ intended to be used for any serious sitemapping activities.** It is a means for me to continue learning how to write idiomatic Go. Mapping a site provides a problem which lends allows me to practise writing Go code, playing with Go's concurrency features, as well as learn about project structure, writing tests, and interacting with 3rd party libraries (Cobra). Further elaboration below.

## Things I've Learnt
Some things this repo includes which are examples of Go language features and patterns which I am learning:

### Concurrency

I make use of channels and sync.WaitGroup to run multiple goroutines, as well as using a buffered channel to implement a concurrency "limiter".
  * See [crawler.go](sitemapper/internal/crawler.go) for 3 crawl engine implementations:
    * **Synchronous**: recursively visits extracted URLs one URL at a time up to a specified tree depth.
    * **Concurrent**: recursively visits extracted URLs up to a specified tree depth, with each visit happening concurrently. A WaitGroup is used to monitor for crawl completion.
    * **Concurrent Limited**: recursively visits extracted URLs up to a specified tree depth, with each visit happening concurrently, with a limit to the number of concurrent visits. If a crawl is attempted when the concurrency limit has been reached the code waits for a random amount of time before attempting to execute again. A WaitGroup is used to monitor for crawl completion.
      * See [limiter.go](sitemapper/internal/limiter.go)
  * See [sitemap.go](sitemapper/internal/sitemap.go) for use of a `sync.RWMutex` to manage concurrent access to an internal map data structure.

### Interfaces

[crawler.go](sitemapper/internal/crawler.go) provides three different implementations of a `Run` method, defined in the `CrawlEngine` interface, for the three different concurrency modes featured by SiteMapper. Commandline options parsed by [root.go](sitemapper/cmd/root.go) determine which implementation is used at runtime.

### Standard Library Interfaces

[sitemap.go](sitemapper/internal/sitemap.go) includes the `SiteMap` struct which has an implementation of `io.WriteTo`, allowing the sitemap contents to be written to anything that meets the `io.Writer` interface. Additionally the implementation of the `WriteTo` method requires an implementation of a custom `io.Writer` `Write` function so that the number of bytes written can be returned.

### HTTP requests
[crawler.go](sitemapper/internal/crawler.go) includes a few lines of code where `http.Get` is used and the response inspected.

### JSON

The output of SiteMapper is JSON written to stdout. [sitemap.go](sitemapper/internal/sitemap.go) includes a custom `JSONMarshal` function which provides a cleaner mapping of the internal sitemap structure to a more readable dictionary of string arrays.

### Third Party Packages

* I'm using [Cobra](https://github.com/spf13/cobra) to parse and map CLI arguments to application features.
* I'm using [Is](https://github.com/matryer/is) to provide basic lightweight test assertions.

### Tests

* Table tests: Many of the tests in [crawler_test.go](sitemapper/internal/crawler_test.go) make use of table tests to cover a range of inputs and expected outputs.
* testdata: [crawler_test.go](sitemapper/internal/crawler_test.go) uses files in the testdata folder to store expected values.

## Build

A makefile is included to facilitate build and test activities. To build the project run:

```shell
make build-standalone
```

## Test

To run the available tests, issue the following command:

```shell
make test
```

Note that the above command will run tests with code coverage enabled.

## Usage

```shell
$ ./sm -h                                                                                                                                                                                                                                                                                 *[main]
Crawls from a start URL and writes a JSON based sitemap to stdout

Usage:
  sm [flags]

Flags:
  -d, --depth int     Specify crawl depth (default 1)
  -h, --help          help for sitemapper
  -l, --limit int     Specify max concurrent crawl tasks for limited mode (default 10)
  -m, --mode string   Specify mode: synchronous, concurrent, limited (default "concurrent")
  -s, --site string   Site to crawl, including http scheme

```

### Examples

#### Concurrent crawl of https://dinofizzotti.com with depth 1

```shell
./sm -s https://dinofizzotti.com
```

#### Synchronous crawl of https://dinofizzotti.com with depth 3

```shell
./sm -s https://dinofizzotti.com -d 3 --mode synchronous
```

#### Concurrent Limited crawl of https://dinofizzotti.com with depth 3 and concurrency limit 5

```shell
./sm -s https://dinofizzotti.com -d 3 --mode limited -l 5
```
