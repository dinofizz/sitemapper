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

type Visitor interface {
	Run()
	Visit(u, root, parent *url.URL, d int)
}

type MapperClient struct {
	V Visitor
}

type SynchronousVisitor struct {
	sm    *SiteMap
	max   int
	start *url.URL
}

type ConcurrentVisitor struct {
	SynchronousVisitor
	WG sync.WaitGroup
}

type ConcurrentLimitedVisitor struct {
	ConcurrentVisitor
	limiter *Limiter
}

func NewSynchronousVisitor(sitemap *SiteMap, max int, start *url.URL) *SynchronousVisitor {
	return &SynchronousVisitor{sm: sitemap, max: max, start: start}
}
func NewConcurrentVisitor(sitemap *SiteMap, max int, start *url.URL) *ConcurrentVisitor {
	return &ConcurrentVisitor{SynchronousVisitor: SynchronousVisitor{sm: sitemap, max: max, start: start}}
}

func NewConcurrentLimitedVisitor(sitemap *SiteMap, max int, start *url.URL, limiter *Limiter) *ConcurrentLimitedVisitor {
	return &ConcurrentLimitedVisitor{
		ConcurrentVisitor: ConcurrentVisitor{
			SynchronousVisitor: SynchronousVisitor{
				sm:    sitemap,
				max:   max,
				start: start,
			},
		},
		limiter: limiter,
	}
}

func (mc MapperClient) Run() {
	mc.V.Run()
}
func (c *SynchronousVisitor) Run() {
	c.Visit(c.start, c.start, c.start, 0)
}

func (c *ConcurrentVisitor) Run() {
	c.Visit(c.start, c.start, c.start, 0)
	c.WG.Wait()
}
func (c *ConcurrentLimitedVisitor) Run() {
	c.Visit(c.start, c.start, c.start, 0)
	c.WG.Wait()
}

func (c *SynchronousVisitor) Visit(u *url.URL, root *url.URL, parent *url.URL, d int) {
	urls, done := getLinks(u, root, parent, d, c.max, c.sm)
	if done {
		return
	}
	d++

	for _, urlLink := range urls {
		c.Visit(urlLink, root, u, d)
	}
}

func (c *ConcurrentVisitor) Visit(u *url.URL, root *url.URL, parent *url.URL, d int) {
	urls, done := getLinks(u, root, parent, d, c.max, c.sm)
	if done {
		return
	}
	d++

	for _, urlLink := range urls {
		c.WG.Add(1)
		go func(urlLink, root, parent *url.URL, d int) {
			defer c.WG.Done()
			c.Visit(urlLink, root, parent, d)
		}(urlLink, root, u, d)
	}
}

func (c *ConcurrentLimitedVisitor) Visit(u *url.URL, root *url.URL, parent *url.URL, d int) {
	urls, done := getLinks(u, root, parent, d, c.max, c.sm)
	if done {
		return
	}
	d++

	for _, urlLink := range urls {
		c.WG.Add(1)
		go func(urlLink, root, parent *url.URL, d int) {
			defer c.WG.Done()
			retries := 0
			for {
				err := c.limiter.RunFunc(func() {
					c.Visit(urlLink, root, parent, d)
				})
				if err != nil {
					n := rand.Intn(500) // n will be between 0 and 10
					log.Printf("task limited for URL %s, sleeping for %d millisecconds\n", urlLink.String(), n)
					time.Sleep(time.Duration(n) * time.Millisecond)
					retries++
				} else {
					break
				}
			}
		}(urlLink, root, u, d)
	}
}

func getLinks(u *url.URL, root *url.URL, parent *url.URL, d int, max int, sm *SiteMap) ([]*url.URL, bool) {
	if max == d {
		return nil, true
	}

	if urls, exists := sm.GetUrls(u); exists == true {
		//log.Printf("ignoring %s as we already have it", u.String())
		return urls, false
	}

	sm.AddUrl(u)
	log.Printf("visiting URL %s at depth %d with parent %s", u.String(), d, parent.String())

	html, err := getHtml(u)
	if err != nil {
		log.Printf("error retrieving HTML for URL %s: %s", u.String(), err)
	}
	links := extractLinks(html)
	urls := cleanLinks(links, root, u)
	if len(urls) > 0 {
		sm.UpdateUrlWithLinks(u, urls)
	}

	//d++
	//if max == d {
	//	return 0, nil, true
	//}
	return urls, false
}

func cleanLinks(links []string, root *url.URL, parent *url.URL) []*url.URL {
	cLinks := make([]*url.URL, 0)

	for _, link := range links {

		l, err := url.Parse(link)
		if err != nil {
			log.Printf("error parsing link %s", link)
			continue
		}

		if l.Scheme != "" && l.Scheme != "http" && l.Scheme != "https" {
			log.Printf("ignorng scheme %s in link %s", l.Scheme, link)
		}

		if l.Host == "" && (l.Path == "" || l.Path == "/") {
			log.Printf("Ignoring link %s", link)
			continue
		}

		var urlLink *url.URL

		if l.Host == "" && strings.HasPrefix(l.Path, "/") {
			urlLink, err = url.Parse(fmt.Sprintf("%s://%s%s", root.Scheme, root.Host, l.String()))
			if err != nil {
				log.Printf("error parsing link %s", link)
				continue
			}
		} else if l.Host == "" && l.Path != "" {
			newPath := path.Join(parent.Path, l.Path)
			s := fmt.Sprintf("%s://%s/%s", parent.Scheme, parent.Host, newPath)
			urlLink, err = url.Parse(s)
			if err != nil {
				log.Printf("error parsing link %s", link)
				continue
			}
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
	re := regexp.MustCompile("<a\\s+(?:[^>]*?\\s+)?href=(\\S+?)[\\s>]")
	matches := re.FindAllStringSubmatch(html, -1)
	if matches == nil || len(matches) == 0 {
		return nil
	}

	lm := make(map[string]struct{}, 0)
	links := make([]string, 0)

	for _, v := range matches {
		m := v[1]
		m = strings.Trim(m, "\"'")
		m = strings.TrimSpace(m)
		if _, ok := lm[m]; ok == false {
			lm[m] = struct{}{}
			links = append(links, m)
		}
	}

	return links
}
