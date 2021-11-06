package internal

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

type Crawler struct {
	sm   *SiteMap
	WG   sync.WaitGroup
	sync bool
}

func NewCrawler(sitemap *SiteMap, synchronous bool) *Crawler {
	return &Crawler{sm: sitemap, sync: synchronous}
}

func (c *Crawler) Visit(u *url.URL, parent *url.URL, d, max int) {
	if d == max {
		return
	}

	if c.sm.UrlExists(u) == true {
		log.Printf("ignoring %s as we already have it", u.String())
		return
	}

	log.Printf("visiting URL %s", u.String())
	c.sm.AddUrl(u)

	html, err := c.GetHtml(u)
	if err != nil {
		log.Printf("error retrieving HTML for URL %s: %s", u.String(), err)
	}
	links := c.FindLinks(html)
	urls := c.CleanLinks(links, parent)
	if len(urls) > 0 {
		c.sm.UpdateUrlWithLinks(u, urls)
	}
	d++
	for _, urlLink := range urls {
		if c.sync == true {
			c.Visit(urlLink, parent, d, max)
		} else {
			c.WG.Add(1)
			go func(urlLink, parent *url.URL, d, max int) {
				defer c.WG.Done()
				c.Visit(urlLink, parent, d, max)
			}(urlLink, parent, d, max)
		}
	}
}

func (c *Crawler) CleanLinks(links []string, u *url.URL) []*url.URL {
	cleanLinks := make([]*url.URL, 0)

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
			urlLink, err = url.Parse(u.String() + l.String())
			if err != nil {
				log.Printf("error parsing link %s", link)
				continue
			}
		}

		if strings.Contains(l.Host, u.Host) {
			urlLink, err = url.Parse(l.String())
			if err != nil {
				log.Printf("error parsing link %s", link)
				continue
			}
		}

		if urlLink != nil {
			cleanLinks = append(cleanLinks, urlLink)
		}
	}

	return cleanLinks
}

func (c *Crawler) GetHtml(u *url.URL) (string, error) {
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

func (c *Crawler) FindLinks(html string) []string {
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
