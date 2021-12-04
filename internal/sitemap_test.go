package internal

import (
	"encoding/json"
	is2 "github.com/matryer/is"
	"testing"
)

func TestSiteMap_AddUrl(t *testing.T) {
	sm := NewSiteMap()
	sm.AddUrl("https://www.example.com")

	is := is2.New(t)

	is.Equal(sm.sitemap["https://www.example.com"], LinkMap{})
}

func TestSiteMap_UpdateUrlWithLinks(t *testing.T) {
	sm := NewSiteMap()
	u := "https://www.example.com"
	sm.AddUrl(u)

	links := []string{"https://link.one/", "https://link.two"}

	is := is2.New(t)
	sm.UpdateUrlWithLinks(u, links)

	expectedMap := LinkMap{
		"https://link.one": "https://link.one/",
		"https://link.two": "https://link.two",
	}

	is.Equal(sm.sitemap["https://www.example.com"], expectedMap)
}

func TestSiteMap_Read(t *testing.T) {
	sm := NewSiteMap()
	u := "https://www.example.com"
	links := []string{"https://link.one/", "https://link.two"}
	expectedMap := map[string][]string{
		"https://www.example.com": []string{
			"https://link.one/",
			"https://link.two",
		},
	}
	is := is2.New(t)
	var output map[string][]string

	sm.AddUrl(u)
	sm.UpdateUrlWithLinks(u, links)

	err := json.NewDecoder(sm).Decode(&output)
	if err != nil {
		t.Fatal(err)
	}
	is.Equal(output, expectedMap)
}
