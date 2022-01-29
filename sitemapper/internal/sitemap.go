package sitemap

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
)

// links is a type definition which is used internally as a list of links found at a parent URL.
// The use of a map ensures that we don't write duplicate entries, and negates the need for searching a slice.
// TODO: rethink this structure, not sure if I need the values AND the keys.
type links map[string]string
type linkMap map[string]links

// A SiteMap is the data structure used to store a list of links found at crawled URLs. A sync.RWMutex provides
// access control to the internal map.
type SiteMap struct {
	mutex   sync.RWMutex
	sitemap linkMap
}

// NewSiteMap returns an SiteMap instance with an empty sitemap map, ready for URLs and links to be added.
func NewSiteMap() *SiteMap {
	return &SiteMap{sitemap: map[string]links{}}
}

// GetLinks returns the slice of links available for a given URL key. If the URL exists in the internal map the links
// are returned to the caller along with a boolean with a value of true.
// If the key is not found in the map a nil slice is returned along with a boolean value of false.
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

// AddURL adds an entry to the internal map for a given URL and initialises the list of links with an empty map.
func (sm *SiteMap) AddURL(u string) {
	s := strings.TrimSuffix(u, "/")
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.sitemap[s] = links{}
}

// UpdateURLWithLinks associates the provided slice of links with the given parent URL.
func (sm *SiteMap) UpdateURLWithLinks(u string, newLinks []string) {
	s := strings.TrimSuffix(u, "/")
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	linkMap := sm.sitemap[s]

	for _, nl := range newLinks {
		l := strings.TrimSuffix(nl, "/")
		linkMap[l] = nl
	}

	sm.sitemap[s] = linkMap
}

// MarshalJSON is provided to aid the marshalling of the internal map structure for a parent URL to a slice of link strings.
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
func (lm links) MarshalJSON() ([]byte, error) {
	l := make([]string, 0)
	for _, v := range lm {
		l = append(l, v)
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
		Count:   len(sm.sitemap),
		Results: &sm.sitemap,
	}

	j, err := json.Marshal(jsm)
	if err != nil {
		return nil, err
	}

	return j, nil
}
