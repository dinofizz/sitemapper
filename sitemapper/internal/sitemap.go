package sitemap

import (
	"encoding/json"
	"sort"
	"sync"
)

// links is a type definition which is used internally as a list of links found at a URL.
// The use of a map ensures that we don't write duplicate entries, and negates the need for searching a slice.
type links map[string]struct{}

// linkMap is the collection of all unique URLs and the links found at each URL
type linkMap map[string]links

// A SiteMap is the data structure used to store a list of links found at crawled URLs. A sync.RWMutex provides
// access control to the internal map.
type SiteMap struct {
	mutex sync.RWMutex
	lm    linkMap
}

// NewSiteMap returns an SiteMap instance with an empty sitemap map, ready for URLs and links to be added.
func NewSiteMap() *SiteMap {
	return &SiteMap{lm: linkMap{}}
}

// GetLinks returns the slice of links available for a given URL key. If the URL exists in the internal map the links
// are returned to the caller along with a boolean with a value of true.
// If the key is not found in the map a nil slice is returned along with a boolean value of false.
func (sm *SiteMap) GetLinks(u string) ([]string, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	urlMap, exists := sm.lm[u]
	if !exists {
		return nil, false
	}

	var urls []string

	for k := range urlMap {
		urls = append(urls, k)
	}

	return urls, exists
}

// AddURL adds an entry to the internal map for a given URL and initialises the list of links with an empty map.
func (sm *SiteMap) AddURL(u string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.lm[u] = links{}
}

// UpdateURLWithLinks associates the provided slice of links with the given parent URL.
func (sm *SiteMap) UpdateURLWithLinks(u string, newLinks []string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	linkMap := sm.lm[u]

	for _, nl := range newLinks {
		linkMap[nl] = struct{}{}
	}

	sm.lm[u] = linkMap
}

// MarshalJSON is provided to aid the marshalling of the internal linkMap structure to a more JSON friendly format
func (lm *linkMap) MarshalJSON() ([]byte, error) {
	type urlLinks struct {
		URL   string
		Links links
	}

	urls := make([]urlLinks, 0)

	for k, v := range *lm {
		ul := &urlLinks{URL: k, Links: v}
		urls = append(urls, *ul)
	}

	j, err := json.Marshal(urls)
	if err != nil {
		return nil, err
	}

	return j, nil
}

// MarshalJSON is provided to aid the marshalling of the internal map structure for a parent URL to a slice of link strings.
func (ls links) MarshalJSON() ([]byte, error) {
	l := make([]string, 0)
	for k := range ls {
		l = append(l, k)
	}

	sort.Strings(l)
	j, err := json.Marshal(l)
	if err != nil {
		return j, err
	}

	return j, nil
}

func (sm *SiteMap) MarshalJSON() ([]byte, error) {

	jsm := struct {
		Count   int
		Results *linkMap
	}{
		Count:   len(sm.lm),
		Results: &sm.lm,
	}

	j, err := json.Marshal(jsm)
	if err != nil {
		return nil, err
	}

	return j, nil
}
