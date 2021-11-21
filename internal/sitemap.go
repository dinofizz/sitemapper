package internal

import (
	"encoding/json"
	"fmt"
	"log"
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

func (sm *SiteMap) Dump() {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	tempMap := map[string][]string{}

	for k := range sm.sitemap {

		linkMap := sm.sitemap[k]
		links := make([]string, 0)
		for kk := range linkMap {
			links = append(links, kk)
		}

		sort.Strings(links)
		tempMap[k] = links
	}

	j, err := json.Marshal(tempMap)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
	} else {
		fmt.Println(string(j))
	}
	log.Println("Number of crawled links: ", len(sm.sitemap))
}
