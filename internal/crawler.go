package internal

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
)

type CrawlEngine interface {
	Run()
	crawl(u, root, parent *url.URL, depth int)
}

type Crawler struct {
	C CrawlEngine
}

type SynchronousCrawlEngine struct {
	sm       *SiteMap
	maxDepth int
	start    *url.URL
}

type ConcurrentCrawlEngine struct {
	SynchronousCrawlEngine
	WG sync.WaitGroup
}

type ConcurrentLimitedCrawlEngine struct {
	ConcurrentCrawlEngine
	limiter *Limiter
}

func NewSynchronousCrawlEngine(sitemap *SiteMap, maxDepth int, start *url.URL) *SynchronousCrawlEngine {
	return &SynchronousCrawlEngine{sm: sitemap, maxDepth: maxDepth, start: start}
}
func NewConcurrentCrawlEngine(sitemap *SiteMap, maxDepth int, start *url.URL) *ConcurrentCrawlEngine {
	return &ConcurrentCrawlEngine{SynchronousCrawlEngine: SynchronousCrawlEngine{sm: sitemap, maxDepth: maxDepth, start: start}}
}

func NewConcurrentLimitedCrawlEngine(sitemap *SiteMap, maxDepth int, start *url.URL, limiter *Limiter) *ConcurrentLimitedCrawlEngine {
	return &ConcurrentLimitedCrawlEngine{
		ConcurrentCrawlEngine: ConcurrentCrawlEngine{
			SynchronousCrawlEngine: SynchronousCrawlEngine{
				sm:       sitemap,
				maxDepth: maxDepth,
				start:    start,
			},
		},
		limiter: limiter,
	}
}

func (mc Crawler) Run() {
	mc.C.Run()
}
func (c *SynchronousCrawlEngine) Run() {
	c.crawl(c.start, c.start, c.start, 0)
}

func (c *ConcurrentCrawlEngine) Run() {
	c.crawl(c.start, c.start, c.start, 0)
	c.WG.Wait()
}
func (c *ConcurrentLimitedCrawlEngine) Run() {
	c.crawl(c.start, c.start, c.start, 0)
	c.WG.Wait()
}

func (c *SynchronousCrawlEngine) crawl(u, root, parent *url.URL, depth int) {
	if c.maxDepth == depth {
		return
	}
	urls := getLinks(u, root, parent, depth, c.sm)
	depth++

	for _, urlLink := range urls {
		c.crawl(urlLink, root, u, depth)
	}
}

func (c *ConcurrentCrawlEngine) crawl(u, root, parent *url.URL, depth int) {
	if c.maxDepth == depth {
		return
	}
	urls := getLinks(u, root, parent, depth, c.sm)
	depth++

	for _, urlLink := range urls {
		c.WG.Add(1)
		go func(urlLink, root, parent *url.URL, d int) {
			defer c.WG.Done()
			c.crawl(urlLink, root, parent, d)
		}(urlLink, root, u, depth)
	}
}

func (c *ConcurrentLimitedCrawlEngine) crawl(u, root, parent *url.URL, depth int) {
	if c.maxDepth == depth {
		return
	}
	urls := getLinks(u, root, parent, depth, c.sm)
	depth++

	for _, urlLink := range urls {
		c.WG.Add(1)
		go func(urlLink, root, parent *url.URL, d int) {
			defer c.WG.Done()
			retries := 0
			for {
				err := c.limiter.RunFunc(func() {
					c.crawl(urlLink, root, parent, d)
				})
				if err != nil {
					n := rand.Intn(500) // n will be between 0 and 10
					log.Printf("task limited for URL %s, sleeping for %depth millisecconds\n", urlLink.String(), n)
					time.Sleep(time.Duration(n) * time.Millisecond)
					retries++
				} else {
					break
				}
			}
		}(urlLink, root, u, depth)
	}
}

func getLinks(url, root, parent *url.URL, depth int, sm *SiteMap) []*url.URL {
	if urls, exists := sm.GetUrls(url); exists {
		return urls
	}

	sm.AddUrl(url)
	log.Printf("visiting URL %s at depth %d with parent %s", url.String(), depth, parent.String())

	html, err := getHtml(url)
	if err != nil {
		log.Printf("error retrieving HTML for URL %s: %s", url.String(), err)
	}
	links := extractLinks(html)
	urls := cleanLinks(links, root, url)
	if len(urls) > 0 {
		sm.UpdateUrlWithLinks(url, urls)
	}

	return urls
}

func cleanLinks(links []string, root *url.URL, parent *url.URL) []*url.URL {
	var cLinks []*url.URL

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

		if l.Host == "" && strings.HasPrefix(l.Path, "/") {
			urlLink = &url.URL{Host: root.Host, Path: l.Path, Scheme: root.Scheme}
		} else if l.Host == "" && l.Path != "" {
			newPath := path.Join(parent.Path, l.Path)
			urlLink = &url.URL{Host: parent.Host, Path: newPath, Scheme: parent.Scheme}
		} else if strings.Contains(l.Host, root.Host) {
			urlLink = &url.URL{Host: l.Host, Path: l.Path, Scheme: l.Scheme}
		}

		if urlLink != nil {
			cLinks = append(cLinks, urlLink)
		}
	}

	return cLinks
}

func getHtml(u *url.URL) (string, error) {
	resp, err := http.Get(u.String())
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received HTTP response code %d for site %s", resp.StatusCode, u)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("error reading response body")
	}
	return string(b), nil
}

func extractLinks(html string) []string {
	re := regexp.MustCompile(`<a\s+(?:[^>]*?\s+)?href=(\S+?)[\s>]`)
	matches := re.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}

	lm := make(map[string]struct{})
	links := make([]string, 0)

	for _, v := range matches {
		m := v[1]
		m = strings.Trim(m, "\"'")
		m = strings.TrimSpace(m)
		if _, ok := lm[m]; !ok {
			lm[m] = struct{}{}
			links = append(links, m)
		}
	}

	return links
}
