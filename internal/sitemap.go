package internal

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

type SiteMap struct {
	l       sync.RWMutex
	sitemap map[string]map[string]*url.URL
	//sitemap map[string][]*url.URL
}

func NewSiteMap() *SiteMap {
	return &SiteMap{sitemap: map[string]map[string]*url.URL{}}
	//return &SiteMap{sitemap: map[string][]*url.URL{}}
}

func (sm *SiteMap) GetUrls(u *url.URL) ([]*url.URL, bool) {
	s := strings.TrimSuffix(u.String(), "/")
	sm.l.RLock()
	defer sm.l.RUnlock()
	urlmap, exists := sm.sitemap[s]
	if exists == false {
		return nil, false
	}

	var urls []*url.URL

	for _, urlv := range urlmap {
		urls = append(urls, urlv)
	}

	return urls, exists
}

func (sm *SiteMap) AddUrl(u *url.URL) {
	s := strings.TrimSuffix(u.String(), "/")
	sm.l.Lock()
	defer sm.l.Unlock()
	sm.sitemap[s] = map[string]*url.URL{}
}

func (sm *SiteMap) UpdateUrlWithLinks(u *url.URL, newLinks []*url.URL) {
	s := strings.TrimSuffix(u.String(), "/")
	sm.l.Lock()
	defer sm.l.Unlock()
	linkMap := sm.sitemap[s]

	for _, nl := range newLinks{
		s := strings.TrimSuffix(nl.String(), "/")
		linkMap[s] = nl
	}

	sm.sitemap[s] = linkMap
}

func (sm *SiteMap) Print() {
	sm.l.RLock()
	defer sm.l.RUnlock()
	var num int

	var keys []string

	for k, _ := range sm.sitemap {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for uniqueIndex, k := range keys {
		fmt.Printf("%d ### %s : %d\n", uniqueIndex+1, k, len(sm.sitemap[k]))

		linkMap := sm.sitemap[k]
		var links []string

		for kk, _ := range linkMap {
			links = append(links, kk)
		}

		sort.Strings(links)
		for linkIndex, l := range links {
			fmt.Println(linkIndex+1, l)
		}
		fmt.Println("")
	}

	fmt.Println("Unique links: ", len(sm.sitemap))
	fmt.Println("Total number of links discovered: ", num)
}