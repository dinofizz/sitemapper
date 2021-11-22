package internal

import (
	"github.com/matryer/is"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_extractLinks(t *testing.T) {
	html := `
<!DOCTYPE html>
<html>
  <head>
	<meta charset="utf-8">
	<title>index</title>
  </head>
  <body>
	<p>index</p>
	<a class="some-class" href="/aubergine">/aubergine</a>
	<a href="biscuit/pomegranate.html" class="foo-class">biscuit/pomegranate.html</a>
	<a id="id-link" href="tomato.html">tomato</a>
	<a href="/" class="another-class">root</a>
  </body>
</html>`

	is := is.New(t)
	links := extractLinks(html)
	is.Equal(len(links), 4)
	is.Equal(links[0], "/aubergine")
	is.Equal(links[1], "biscuit/pomegranate.html")
	is.Equal(links[2], "tomato.html")
	is.Equal(links[3], "/")
}

func Test_getHtml(t *testing.T) {
	html := `
<!DOCTYPE html>
<html>
  <head>
	<meta charset="utf-8">
	<title>index</title>
  </head>
  <body>
	<p>index</p>
	<a class="some-class" href="/aubergine">/aubergine</a>
	<a href="biscuit/pomegranate.html" class="foo-class">biscuit/pomegranate.html</a>
	<a id="id-link" href="tomato.html">tomato</a>
	<a href="/" class="another-class">root</a>
  </body>
</html>`
	is := is.New(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write([]byte(html))
		if err != nil {
			log.Fatal(err.Error())
		}
	}))

	defer srv.Close()

	result, _, _ := getHtml(srv.URL)

	is.Equal(result, html)
}
