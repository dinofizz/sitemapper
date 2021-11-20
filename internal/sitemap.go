package internal

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type SiteMap struct {
	mutex   sync.RWMutex
	sitemap map[string]map[string]string
}

func NewSiteMap() *SiteMap {
	return &SiteMap{sitemap: map[string]map[string]string{}}
}

func (sm *SiteMap) GetLinks(u string) ([]string, bool) {
	s := strings.TrimSuffix(u, "/")
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	urlMap, exists := sm.sitemap[s]
	if !exists {
		return nil, false
	}

	var urls []string

	for _, v := range urlMap {
		urls = append(urls, v)
	}

	return urls, exists
}

func (sm *SiteMap) AddUrl(u string) {
	s := strings.TrimSuffix(u, "/")
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.sitemap[s] = map[string]string{}
}

func (sm *SiteMap) UpdateUrlWithLinks(u string, newLinks []string) {
	s := strings.TrimSuffix(u, "/")
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	linkMap := sm.sitemap[s]

	for _, nl := range newLinks{
		s := strings.TrimSuffix(nl, "/")
		linkMap[s] = nl
	}

	sm.sitemap[s] = linkMap
}

func (sm *SiteMap) Print() {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var keys []string

	for k := range sm.sitemap {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for uniqueIndex, k := range keys {
		fmt.Printf("%d ### %s : %d\n", uniqueIndex+1, k, len(sm.sitemap[k]))

		linkMap := sm.sitemap[k]
		var links []string

		for kk := range linkMap {
			links = append(links, kk)
		}

		sort.Strings(links)
		for linkIndex, l := range links {
			fmt.Println(linkIndex+1, l)
		}
		fmt.Println("")
	}

	fmt.Println("Number of crawled links: ", len(sm.sitemap))
}
