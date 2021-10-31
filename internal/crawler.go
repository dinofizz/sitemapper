package internal

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Crawler struct {
	sm *SiteMap
}

func NewCrawler(sitemap *SiteMap) *Crawler {
	return &Crawler{sm: sitemap}
}

func (c *Crawler) Visit(u *url.URL, parent *url.URL, d, max int) {
	if d == max {
		return
	}

	if c.sm.UrlExists(u) == true {
		//fmt.Printf("Ignoring %s as we already have it\n", u.String())
		return
	}

	c.sm.AddUrl(u)

	html, _ := c.GetHtml(u)
	links := c.FindLinks(html)
	urls := c.CleanLinks(links, parent)
	if len(urls) > 0 {
		c.sm.UpdateUrlWithLinks(u, urls)
	}
	d++
	for _, urlLink := range urls {
		c.Visit(urlLink, parent, d, max)
	}
}

func (c *Crawler) PrintLinks(urls []*url.URL) {
	for _, u := range urls {
		fmt.Println(u.String())
	}
}

func (c *Crawler) CleanLinks(links []string, u *url.URL) []*url.URL {
	cleanLinks := make([]*url.URL, 0)

	for _, link := range links {

		l, err := url.Parse(link)
		if err != nil {
			fmt.Printf("error parsing link %s\n", link)
			continue
		}

		if l.Host == "" && (l.Path == "" || l.Path == "/") {
			continue
		}

		if l.Host == "" && strings.HasPrefix(l.Path, "/") {
			urlLink, err := url.Parse(u.String() + l.String())
			if err != nil {
				fmt.Printf("error parsing link %s\n", link)
				continue
			}
			cleanLinks = append(cleanLinks, urlLink)
		}

		if strings.Contains(l.Host, u.Host) {
			urlLink, err := url.Parse(l.String())
			if err != nil {
				fmt.Printf("error parsing link %s\n", link)
				continue
			}
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
	re := regexp.MustCompile("<a\\s+(?:[^>]*?\\s+)?href=\"([^\"]*)\"")
	matches := re.FindAllStringSubmatch(html, -1)
	if matches == nil || len(matches) == 0 {
		return nil
	}

	links := make([]string, 0)

	for _, v := range matches {
		links = append(links, strings.TrimSpace(v[1]))
	}
	return links
}
