package i18n

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/text/language"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeContents struct {
	contents map[string]string
}

func (fs *fakeContents) Reader(path string) (io.Reader, error) {
	r, ok := fs.contents[path]

	if !ok {
		return nil, fmt.Errorf("File %s not found in the fake filesystem", path)
	}

	return strings.NewReader(r), nil
}

type fallbackHandler struct {
}

func (f *fallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Fallback Content"))
}

type fakeTranslators struct {
}

func (t *fakeTranslators) Translator(language.Tag) Translator {
	return &fakeTranslator{}
}

func (t *fakeTranslators) Matcher() language.Matcher {
	return language.NewMatcher([]language.Tag{})
}

type fakeTranslator struct {
}

func (t *fakeTranslator) Translate(s string, args ...interface{}) (string, error) {
	return "trans -> " + s, nil
}

func TestTemplates(t *testing.T) {
	Convey("Test Templates", t, func() {
		fs := &fakeContents{
			contents: map[string]string{
				"/index.i18n.html":            ">> {{translate `Root Index`}}",
				"/some/random/page.i18n.html": "== {{translate `Some Random Page`}}",
			},
		}

		fh := &fallbackHandler{}
		translators := &fakeTranslators{}

		s := httptest.NewServer(Wrap(fh, fs, translators))
		c := &http.Client{}

		Convey("Get non translated content", func() {
			r, err := c.Get(s.URL + "/nontranslated.html")
			So(err, ShouldBeNil)
			content, err := ioutil.ReadAll(r.Body)
			So(string(content), ShouldEqual, "Fallback Content")
		})

		Convey("Get /index.i18n.html from / using default language", func() {
			r, err := c.Get(s.URL + "/")
			So(err, ShouldBeNil)
			content, err := ioutil.ReadAll(r.Body)
			So(string(content), ShouldEqual, ">> trans -> Root Index")
		})

		Convey("Non index page using default language", func() {
			r, err := c.Get(s.URL + "/some/random/page.i18n.html")
			So(err, ShouldBeNil)
			content, err := ioutil.ReadAll(r.Body)
			So(string(content), ShouldEqual, "== trans -> Some Random Page")
		})
	})
}
