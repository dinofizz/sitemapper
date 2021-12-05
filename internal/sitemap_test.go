package sitemap

import (
	"bytes"
	"encoding/json"
	is2 "github.com/matryer/is"
	"testing"
)

func TestSiteMap_AddUrl(t *testing.T) {
	sm := NewSiteMap()
	sm.AddURL("https://www.example.com")

	is := is2.New(t)

	is.Equal(sm.sitemap["https://www.example.com"], links{})
}

func TestSiteMap_UpdateUrlWithLinks(t *testing.T) {
	sm := NewSiteMap()
	u := "https://www.example.com"
	sm.AddURL(u)

	l := []string{"https://link.one/", "https://link.two"}

	is := is2.New(t)
	sm.UpdateURLWithLinks(u, l)

	expectedMap := links{
		"https://link.one": "https://link.one/",
		"https://link.two": "https://link.two",
	}

	is.Equal(sm.sitemap["https://www.example.com"], expectedMap)
}

func TestSiteMap_WriteTo(t *testing.T) {
	sm := NewSiteMap()
	u := "https://www.example.com"
	links := []string{"https://link.one/", "https://link.two"}
	expectedMap := map[string][]string{
		"https://www.example.com": {
			"https://link.one/",
			"https://link.two",
				},
	}
	is := is2.New(t)
	var output map[string][]string

	sm.AddURL(u)
	sm.UpdateURLWithLinks(u, links)
    var b bytes.Buffer
	_, err := sm.WriteTo(&b)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(b.Bytes(), &output)
	if err != nil {
		t.Fatal(err)
	}

	is.Equal(output, expectedMap)
}
