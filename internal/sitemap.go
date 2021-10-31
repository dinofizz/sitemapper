package internal

import (
	"fmt"
	"net/url"
	"sync"
)

type SiteMap struct {
	l       sync.RWMutex
	sitemap map[string][]*url.URL
}

func NewSiteMap() *SiteMap {
	return &SiteMap{sitemap: map[string][]*url.URL{}}
}

func (sm *SiteMap) UrlExists(url *url.URL) bool {
	sm.l.RLock()
	defer sm.l.RUnlock()
	_, exists := sm.sitemap[url.String()]
	return exists
}

func (sm *SiteMap) AddUrl(u *url.URL) {
	sm.l.Lock()
	defer sm.l.Unlock()
	sm.sitemap[u.String()] = make([]*url.URL, 0)
}

func (sm *SiteMap) UpdateUrlWithLinks(u *url.URL, newLinks []*url.URL) {
	sm.l.Lock()
	defer sm.l.Unlock()
	links := sm.sitemap[u.String()]
	links = append(links, newLinks...)
	sm.sitemap[u.String()] = links
}

func (sm *SiteMap) Print() {
	sm.l.RLock()
	defer sm.l.RUnlock()
	for k, v := range sm.sitemap {
		fmt.Println(k, len(v))
	}
}
