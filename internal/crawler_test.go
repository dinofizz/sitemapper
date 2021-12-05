package sitemap

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/matryer/is"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
)

// startHttpServer is based off the solution for HTTP server graceful shutdown from https://stackoverflow.com/a/42533360
func startHttpServer(wg *sync.WaitGroup) *http.Server {
	srv := &http.Server{Addr: ":2015"}
	fs := http.FileServer(http.Dir("../testsite"))

	http.Handle("/", fs)

	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	return srv
}

func TestCrawlEngine_Run(t *testing.T) {
	// Set up the crawlEngine engines we wish to test
	smSce := NewSiteMap()
	sce := NewSynchronousCrawlEngine(smSce, 5, "http://localhost:2015")
	smCce := NewSiteMap()
	cce := NewConcurrentCrawlEngine(smCce, 5, "http://localhost:2015")
	limiter := NewLimiter(1)
	smClce := NewSiteMap()
	clce := NewConcurrentLimitedCrawlEngine(smClce, 5, "http://localhost:2015", limiter)

	// Run a local HTTP server serving static content pointing to the "testsite" directory
	httpServerExitDone := &sync.WaitGroup{}
	httpServerExitDone.Add(1)
	srv := startHttpServer(httpServerExitDone)

	data := []struct {
		name        string
		crawlEngine CrawlEngine
		sitemap     *SiteMap
	}{
		{"Synchronous crawl engine", sce, smSce},
		{"Concurrent crawl engine", cce, smCce},
		{"Concurrent Limited crawl engine", clce, smClce},
	}

	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			d.crawlEngine.Run()
			// NOTE: The expected output represents the JSON that is printed out
			// when running the program. The internal sitemap data structure is slightly different
			// in that the list of links for a parent link is a map and not a slice
			// So in the test below we marshal out from the internal sitemap to a byte slice
			// (there is a custom JSONMarshal for the map to slice conversion).
			// This is then unmarshalled into a map of slices, which we can then more easily compare
			// with the expected output which is stored in JSON file.

			out, err := json.Marshal(d.sitemap.sitemap)
			if err != nil {
				t.Fatal(err)
			}
			var actualResults map[string][]string
			err = json.Unmarshal(out, &actualResults)
			if err != nil {
				t.Fatal(err)
			}

			resultsFile, err := os.Open("testdata/integration_test_results.json")
			if err != nil {
				t.Fatal(err)
			}

			var expectedResults map[string][]string
			err = json.NewDecoder(resultsFile).Decode(&expectedResults)
			if err != nil {
				t.Fatal(err)
			}

			is := is.New(t)

			is.True(len(d.sitemap.sitemap) == len(expectedResults))
			is.Equal(actualResults, expectedResults)
		})
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	httpServerExitDone.Wait()
}

func Test_extractLinks(t *testing.T) {

	data := []struct {
		name     string
		testfile string
		links    []string
	}{
		{"four links", "testdata/fourlinks.html", []string{"/aubergine", "biscuit/pomegranate.html", "tomato.html", "/"}},
		{"no links", "testdata/nolinks.html", nil},
	}
	is := is.New(t)

	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			file, err := ioutil.ReadFile(d.testfile)
			if err != nil {
				t.Fatal(err)
			}
			html := string(file)
			links := extractLinks(html)

			is.Equal(len(links), len(d.links))
			is.Equal(links, d.links)
		})
	}
}

func Test_getHtml(t *testing.T) {
	data := []struct {
		name         string
		responseBody string
		expectedBody string
		statusCode   int
		err          error
	}{
		{"200 success", "expected body", "expected body", 200, nil},
		{"200 success empty body", "", "", 200, nil},
		{"404 error", "", "", 404, errors.New("received HTTP response code 404 for site http://127.0.0.1:")},
		{"500 error", "expected body", "", 500, errors.New("received HTTP response code 500 for site http://127.0.0.1:")},
		{"server error", "", "", 500, errors.New("received HTTP response code 500 for site http://127.0.0.1:")},
	}

	is := is.New(t)

	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			var sUrl string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(d.statusCode)
				_, err := w.Write([]byte(d.responseBody))
				if err != nil {
					log.Fatal(err.Error())
				}
			}))

			sUrl = srv.URL
			defer srv.Close()

			result, requestUrl, err := getHTML(sUrl)

			is.Equal(result, d.expectedBody)
			is.Equal(requestUrl.String(), sUrl)
			if d.err != nil {
				is.True(strings.Contains(err.Error(), d.err.Error()))
			} else {
				is.NoErr(err)
			}
		})
	}
}

func Test_getHtml_BadServer(t *testing.T) {
	is := is.New(t)
	sUrl := "http://badserver"
	result, requestUrl, err := getHTML(sUrl)

	is.Equal(result, "")
	is.Equal(requestUrl, nil)
	var urlError *net.DNSError
	is.True(errors.As(err, &urlError))
}

func Test_cleanLinks(t *testing.T) {
	data := []struct {
		name          string
		root          string
		parent        string
		inputLinks    []string
		expectedLinks []string
	}{
		{"parent trailing slash", "https://example.com", "https://example.com/parent/",
			[]string{
				"relative/link/index.html",
				"/absolute/index.html",
				"/",
				"https://anotherhost.com/link.html",
				"mailto://test@email.com",
				"https://example.com/index.html#anchor",
			},
			[]string{
				"https://example.com/parent/relative/link/index.html",
				"https://example.com/absolute/index.html",
				"https://example.com/index.html",
			},
		},
		{"parent index.html", "https://example.com", "https://www.example.com/parent/index.html",
			[]string{
				"relative/link/index.html",
				"/absolute/index.html",
				"/",
				"https://anotherhost.com/link.html",
				"mailto://test@email.com",
				"https://example.com/index.html#anchor",
			},
			[]string{
				"https://www.example.com/parent/relative/link/index.html",
				"https://www.example.com/absolute/index.html",
				"https://example.com/index.html",
			},
		},
		{"bad link", "https://example.com", "https://example.com/parent/index.html",
			[]string{
				string([]byte{0x7f}),
			},
			nil,
		}, {"bad root", string([]byte{0x7f}), "https://example.com/parent/index.html",
			[]string{"https://example.com/link.html"},
			nil,
		},
	}
	is := is.New(t)

	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			p, err := url.Parse(d.parent)
			if err != nil {
				t.Fatalf(err.Error())
			}
			cLinks := cleanLinks(d.inputLinks, d.root, p)
			is.Equal(len(cLinks), len(d.expectedLinks))
			for i, l := range cLinks {
				is.Equal(l, d.expectedLinks[i])
			}
		})
	}
}

func Test_getLinks(t *testing.T) {
	data := []struct {
		name          string
		responseBody  string
		expectedLinks []string
		statusCode    int
		existingLinks bool
	}{
		{"existing links returned", `<a href="https://example.com">link</a>`, []string{"https://example.com"}, 200, true},
		{"links returned", `<a href="https://example.com">link</a>`, []string{"https://example.com"}, 200, false},
		{"500 error", `<a href="https://example.com">link</a>`, nil, 500, false},
		{"empty body", "", nil, 200, false},
		{"no links", "no links here", nil, 200, false},
		{"no clean links", `<a href="/">link</a>`, nil, 200, false},
	}

	is := is.New(t)

	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			var sUrl string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(d.statusCode)
				_, err := w.Write([]byte(d.responseBody))
				if err != nil {
					log.Fatal(err.Error())
				}
			}))

			sUrl = srv.URL
			defer srv.Close()

			root := "https://example.com"
			parent := "https://example.com/foo/"
			depth := 1
			sm := NewSiteMap()
			if d.existingLinks {
				sm.AddURL(srv.URL)
				sm.UpdateURLWithLinks(srv.URL, d.expectedLinks)
			}

			links := getLinks(sUrl, root, parent, depth, sm)

			is.Equal(links, d.expectedLinks)
		})
	}
}
