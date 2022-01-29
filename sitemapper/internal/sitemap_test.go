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
		"https://link.one/": "https://link.one/",
		"https://link.two":  "https://link.two",
	}

	is.Equal(sm.sitemap["https://www.example.com"], expectedMap)
}

func TestSiteMap_EncodeOutput(t *testing.T) {
	sm := NewSiteMap()
	u := "https://www.example.com"

	type result struct {
		URL   string
		Links []string
	}

	type resultContainer struct {
		Count   int
		Results []result
	}

	var expected = resultContainer{
		Count: 1,
		Results: []result{
			{
				URL: "https://www.example.com",
				Links: []string{
					"https://link.one/",
					"https://link.two",
				},
			},
		},
	}

	is := is2.New(t)
	var actual resultContainer

	sm.AddURL(u)
	sm.UpdateURLWithLinks(u, expected.Results[0].Links)
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	err := enc.Encode(sm)
	is.NoErr(err)
	err = json.Unmarshal(b.Bytes(), &actual)
	is.NoErr(err)
	is.Equal(actual, expected)
}
