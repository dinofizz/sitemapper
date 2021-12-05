package sitemap

import (
	"encoding/json"
	"io"
	"sort"
	"strings"
	"sync"
)

// links is a type definition which is used internally as a list of links found at a parent URL.
// The use of a map ensures that we don't write duplicate entries, and negates the need for searching a slice.
// TODO: rethink this structure, not sure if I need the values AND the keys.
type links map[string]string

// A SiteMap is the data structure used to store a list of links found at crawled URLs. A sync.RWMutex provides
// access control to the internal map.
type SiteMap struct {
	mutex   sync.RWMutex
	sitemap map[string]links
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
func (lm links) MarshalJSON() ([]byte, error) {
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

// CounterWr wraps an io.Writer with a Count variable to allow for our implementation of WriteTo which needs
// to return the number of bytes written.
type CounterWr struct {
	io.Writer
	Count int64
}

// Write implements the io.Writer interface and exists to support our implementation of WriteTo which needs
// to return the number of bytes written.
func (cw *CounterWr) Write(p []byte) (n int, err error) {
	n, err = cw.Writer.Write(p)
	cw.Count += int64(n)
	return n, err
}

// WriteTo implements the io.WriteTo interface and is made available such that the internal map structure can be
// written out to the io.Writer provided.
func (sm *SiteMap) WriteTo(w io.Writer) (n int64, err error) {
	cw := &CounterWr{Count: 0, Writer: w}

	enc := json.NewEncoder(cw)
	if err = enc.Encode(&sm.sitemap); err != nil {
		return cw.Count, err
	}

	return cw.Count, err
}
