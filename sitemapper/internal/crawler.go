// Package sitemap contains the data structures and crawl engine implementations for creating a sitemap.
package sitemap

import (
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

const MAX_BACKOFF_MS int = 500

// CrawlEngine is the interface implemented by the various crawl engines.
type CrawlEngine interface {
	Run()
}

// A SynchronousCrawlEngine recursively visits extracted URLs one URL at a time up to a specified tree depth.
type SynchronousCrawlEngine struct {
	sm       *SiteMap
	maxDepth int
	startURL string
}

// A ConcurrentCrawlEngine recursively visits extracted URLs up to a specified tree depth,
// with each visit happening concurrently. A WaitGroup is used to monitor for crawl completion.
type ConcurrentCrawlEngine struct {
	SynchronousCrawlEngine
	WG sync.WaitGroup
}

// A ConcurrentLimitedCrawlEngine recursively visits extracted URLs up to a specified tree depth,
// with each visit happening concurrently, with a limit to the number of concurrent visits.
// A WaitGroup is used to monitor for crawl completion.
type ConcurrentLimitedCrawlEngine struct {
	ConcurrentCrawlEngine
	limiter *Limiter
}

// NewSynchronousCrawlEngine returns a pointer to an instance of a SynchronousCrawlEngine.
func NewSynchronousCrawlEngine(sitemap *SiteMap, maxDepth int, startURL string) *SynchronousCrawlEngine {
	return &SynchronousCrawlEngine{sm: sitemap, maxDepth: maxDepth, startURL: startURL}
}

// NewConcurrentCrawlEngine returns a pointer to an instance of a ConcurrentCrawlEngine.
func NewConcurrentCrawlEngine(sitemap *SiteMap, maxDepth int, startURL string) *ConcurrentCrawlEngine {
	return &ConcurrentCrawlEngine{SynchronousCrawlEngine: SynchronousCrawlEngine{sm: sitemap, maxDepth: maxDepth, startURL: startURL}}
}

// NewConcurrentLimitedCrawlEngine returns a pointer to an instance of a ConcurrentLimitedCrawlEngine.
func NewConcurrentLimitedCrawlEngine(sitemap *SiteMap, maxDepth int, startURL string, limiter *Limiter) *ConcurrentLimitedCrawlEngine {
	return &ConcurrentLimitedCrawlEngine{
		ConcurrentCrawlEngine: ConcurrentCrawlEngine{
			SynchronousCrawlEngine: SynchronousCrawlEngine{
				sm:       sitemap,
				maxDepth: maxDepth,
				startURL: startURL,
			},
		},
		limiter: limiter,
	}
}

// Run begins the sitemap crawl activity for the SynchronousCrawlEngine.
func (c *SynchronousCrawlEngine) Run() {
	c.crawl(c.startURL, c.startURL, c.startURL, 0)
}

// Run begins the sitemap crawl activity for the ConcurrentCrawlEngine.
func (c *ConcurrentCrawlEngine) Run() {
	c.crawl(c.startURL, c.startURL, c.startURL, 0)
	c.WG.Wait()
}

// Run begins the sitemap crawl activity for the ConcurrentLimitedCrawlEngine.
func (c *ConcurrentLimitedCrawlEngine) Run() {
	rand.Seed(time.Now().Unix())
	c.crawl(c.startURL, c.startURL, c.startURL, 0)
	c.WG.Wait()
}

// crawl is the recursive function which is called for each visit to a specified URL.
// For the SynchronousCrawlEngine, crawl performs a recursive synchronous depth-first traversal.
func (c *SynchronousCrawlEngine) crawl(u, root, parent string, depth int) {
	if c.maxDepth == depth {
		return
	}
	urls, exists := getLinks(u, root, parent, depth, c.sm)
	if exists {
		return
	}
	depth++

	for _, urlLink := range urls {
		c.crawl(urlLink, root, u, depth)
	}
}

// crawl is the recursive function which is called for each visit to a specified URL.
// For the ConcurrentCrawlEngine, crawl performs a recursive concurrent traversal where each URL at each depth
// is crawled in a new goroutine concurrently. A WaitGroup keeps track of the number of goroutines.
func (c *ConcurrentCrawlEngine) crawl(u, root, parent string, depth int) {
	if c.maxDepth == depth {
		return
	}
	urls, exists := getLinks(u, root, parent, depth, c.sm)
	if exists {
		return
	}
	depth++

	for _, urlLink := range urls {
		c.WG.Add(1)
		ul := urlLink
		go func() {
			defer c.WG.Done()
			c.crawl(ul, root, parent, depth)
		}()
	}
}

// crawl is the recursive function which is called for each visit to a specified URL.
// For the ConcurrentLimitedCrawlEngine, crawl performs a recursive concurrent traversal where each URL at each depth
// is crawled in a new goroutine concurrently, with a limited enforcing the maximum number of concurrent goroutines.
// A WaitGroup keeps track of the number of goroutines.
func (c *ConcurrentLimitedCrawlEngine) crawl(u, root, parent string, depth int) {
	if c.maxDepth == depth {
		return
	}
	urls, exists := getLinks(u, root, parent, depth, c.sm)
	if exists {
		return
	}
	depth++

	for _, urlLink := range urls {
		c.WG.Add(1)
		ul := urlLink
		go func() {
			defer c.WG.Done()
			for {
				err := c.limiter.RunFunc(func() {
					c.crawl(ul, root, parent, depth)
				})
				if err != nil {
					n := rand.Intn(MAX_BACKOFF_MS)
					log.Printf("task limited for URL %s, sleeping for %d millisecconds\n", ul, n)
					time.Sleep(time.Duration(n) * time.Millisecond)
				} else {
					break
				}
			}
		}()
	}
}

// getLinks performs a series of tasks, calling into other functions responsible for fetching the HTML,
// extracting any links, and then cleaning the extracted links.
// getLinks returns a slice of strings of relevant and applicable links as related to the parent and root URLs.
func getLinks(url, root, parent string, depth int, sm *SiteMap) ([]string, bool) {
	if urls, exists := sm.GetLinks(url); exists {
		return urls, true
	}

	sm.AddURL(url)
	log.Printf("visiting URL %s at depth %d with parent %s", url, depth, parent)

	content, requestUrl, err := getHTML(url)
	if err != nil {
		log.Printf("error retrieving content for URL %s: %v", url, err)
		return nil, false
	}
	if content == "" {
		return nil, false
	}
	links, err := extractLinks(content)
	if err != nil {
		log.Printf("error extracting links from HTML content for URL %s: %v", url, err)
		return nil, false
	}
	if links == nil {
		return nil, false
	}

	urls := cleanLinks(links, root, requestUrl)
	if len(urls) > 0 {
		sm.UpdateURLWithLinks(url, urls)
	}

	return urls, false
}

// cleanLinks accepts a list of links and applies a set of rules to determine whether the links should be included
// in the sitemap results
// cleanLinks returns a slice of full URLs strings including scheme, host and path for any applicable links.
func cleanLinks(links []string, root string, parentUrl *url.URL) []string {
	var cLinks []string

	for _, link := range links {

		l, err := url.Parse(link)
		if err != nil {
			log.Printf("error parsing link %s", link)
			continue
		}

		if l.Scheme != "" && l.Scheme != "http" && l.Scheme != "https" {
			log.Printf("ignoring scheme %s in link %s", l.Scheme, link)
			continue
		}

		if l.Host == "" && (l.Path == "" || l.Path == "/") {
			log.Printf("ignoring link %s", link)
			continue
		}

		var urlLink *url.URL
		rootUrl, err := url.Parse(root)
		if err != nil {
			log.Printf("error parsing root URL %s", root)
			continue
		}

		switch {
		// Relative URL to host
		case l.Host == "" && strings.HasPrefix(l.Path, "/"):
			urlLink = &url.URL{Host: parentUrl.Host, Path: l.Path, Scheme: rootUrl.Scheme}
		// Relative URL to current parent path, append to current document path
		case l.Host == "" && l.Path != "" && strings.HasSuffix(parentUrl.Path, "/"):
			newPath := path.Join(parentUrl.Path, l.Path)
			urlLink = &url.URL{Host: parentUrl.Host, Path: newPath, Scheme: parentUrl.Scheme}
		// Relative URL to current path, same depth as current document
		case l.Host == "" && l.Path != "":
			li := strings.LastIndex(parentUrl.Path, "/")
			parentPath := parentUrl.Path[:li+1]
			newPath := path.Join(parentPath, l.Path)
			urlLink = &url.URL{Host: parentUrl.Host, Path: newPath, Scheme: parentUrl.Scheme}
		// Absolute link featuring same host as root
		case l.Host == rootUrl.Host:
			urlLink = &url.URL{Host: l.Host, Path: l.Path, Scheme: l.Scheme}
		}

		if urlLink != nil {
			cLinks = append(cLinks, urlLink.String())
		}
	}

	return cLinks
}

// getHTML visits the provided URL and returns any HTML in the response as a string.
func getHTML(u string) (string, *url.URL, error) {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return "", nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", resp.Request.URL, fmt.Errorf("received HTTP response code %d for site %s", resp.StatusCode, u)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", resp.Request.URL, errors.New("error reading response body")
	}
	return string(b), resp.Request.URL, nil
}

// extractLinks applies a regular expression pattern to an HTML string and returns a slice of string links
// Implementation uses the example from the docs: https://pkg.go.dev/golang.org/x/net/html#example-Parse
func extractLinks(content string) ([]string, error) {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	// Using a map to build a set of unique links
	// When we add a new link to the map we also append it to the slice which is returned
	lm := make(map[string]struct{})
	var links []string

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					v := a.Val
					v = strings.TrimSpace(v)
					if _, ok := lm[v]; !ok {
						lm[v] = struct{}{}
						links = append(links, v)
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return links, nil
}
