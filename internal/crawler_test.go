package internal

import (
	"errors"
	"github.com/matryer/is"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
				t.Fail()
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
		createSrv    bool
	}{
		{"200 success", "expected body", "expected body", 200, nil, true},
		{"200 success empty body", "", "", 200, nil, true},
		{"404 error", "", "", 404, errors.New("received HTTP response code 404 for site http://127.0.0.1:"), true},
		{"500 error", "expected body", "", 500, errors.New("received HTTP response code 500 for site http://127.0.0.1:"), true},
		{"server error", "", "", 500, errors.New("received HTTP response code 500 for site http://127.0.0.1:"), false},
	}

	is := is.New(t)

	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			var sUrl string
			if d.createSrv {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(d.statusCode)
					_, err := w.Write([]byte(d.responseBody))
					if err != nil {
						log.Fatal(err.Error())
					}
				}))

				sUrl = srv.URL
				defer srv.Close()
			} else {
				sUrl = "http://badserver"
			}

			result, requestUrl, err := getHtml(sUrl)

			if d.createSrv {

				is.Equal(result, d.expectedBody)
				is.Equal(requestUrl.String(), sUrl)
				if d.err != nil {
					is.True(strings.Contains(err.Error(), d.err.Error()))
				} else {
					is.NoErr(err)
				}
			} else {
				is.Equal(result, d.expectedBody)
				is.Equal(requestUrl, nil)
				var urlError *net.DNSError
				is.True(errors.As(err, &urlError))
			}
		})
	}

}
