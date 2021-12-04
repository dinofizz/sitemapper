package sitemap

import (
	"encoding/json"
	"io"
	"sort"
	"strings"
	"sync"
)

type LinkMap map[string]string

type SiteMap struct {
	mutex   sync.RWMutex
	sitemap map[string]LinkMap
}

func NewSiteMap() *SiteMap {
	return &SiteMap{sitemap: map[string]LinkMap{}}
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

	for _, nl := range newLinks {
		s := strings.TrimSuffix(nl, "/")
		linkMap[s] = nl
	}

	sm.sitemap[s] = linkMap
}

func (lm LinkMap) MarshalJSON() ([]byte, error) {
	links := make([]string, 0)
	for _, v := range lm {
		links = append(links, v)
	}

	sort.Strings(links)
	j, err := json.Marshal(links)
	if err != nil {
		return j, err
	}

	return j, nil
}

type CounterWr struct {
	io.Writer
	Count int64
}

func (cw *CounterWr) Write(p []byte) (n int, err error) {
	n, err = cw.Writer.Write(p)
	cw.Count += int64(n)
	return n, err
}

func (sm *SiteMap) WriteTo(w io.Writer) (n int64, err error) {
	cw := &CounterWr{Count: 0, Writer: w}

	enc := json.NewEncoder(cw)
	if err := enc.Encode(&sm.sitemap); err != nil {
		return 0, err
	}

	return cw.Count, err
}
