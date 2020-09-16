package i18n

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"golang.org/x/text/language"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeFile struct {
	*strings.Reader
	modificationTime time.Time
}

func (f *fakeFile) ModificationTime() time.Time {
	return f.modificationTime
}

type fakeFileContent struct {
	content          string
	modificationTime time.Time
}

type fakeContents struct {
	contents map[string]*fakeFileContent
}

func (fs *fakeContents) Reader(path string) (File, error) {
	r, ok := fs.contents[path]

	if !ok {
		return nil, fmt.Errorf("File %s not found in the fake filesystem", path)
	}

	return &fakeFile{Reader: strings.NewReader(r.content), modificationTime: r.modificationTime}, nil
}

type fallbackHandler struct {
}

func (f *fallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Fallback Content"))
}

type fakeTranslators struct {
}

func (t *fakeTranslators) Translator(tag language.Tag, accessTime time.Time) Translator {
	return &fakeTranslator{translationTime: accessTime}
}

func (t *fakeTranslators) Matcher() language.Matcher {
	return language.NewMatcher([]language.Tag{language.English})
}

type fakeTranslator struct {
	translationTime time.Time
}

func (t *fakeTranslator) Translate(s string, args ...interface{}) (string, error) {
	// Renders the translation time as well for testing the cache reuse
	tt := t.translationTime
	return fmt.Sprintf("trans (%02d:%02d:%02d) -> %s", tt.Hour(), tt.Minute(), tt.Second(), s), nil
}

type fakeNow struct {
	now time.Time
}

func (n *fakeNow) Now() time.Time {
	return n.now
}

func TestTemplates(t *testing.T) {
	Convey("Test Templates", t, func() {
		fs := &fakeContents{
			contents: map[string]*fakeFileContent{
				"/index.i18n.html": {
					modificationTime: testutil.MustParseTime(`2000-01-01 00:00:03 +0000`),
					content:          ">> {{translate `Root Index`}}",
				},
				"/some/random/page.i18n.html": {
					modificationTime: testutil.MustParseTime(`2000-01-01 00:00:04 +0000`),
					content:          "== {{translate `Some Random Page`}}",
				},
			},
		}

		fh := &fallbackHandler{}
		translators := &fakeTranslators{}
		now := fakeNow{now: testutil.MustParseTime(`2000-01-01 00:00:10 +0000`)}

		s := httptest.NewServer(Wrap(fh, fs, translators, &now))
		c := &http.Client{}

		Convey("Get non translated content", func() {
			r, err := c.Get(s.URL + "/nontranslated.html")
			So(err, ShouldBeNil)
			content, err := ioutil.ReadAll(r.Body)
			So(string(content), ShouldEqual, "Fallback Content")
		})

		Convey("Error on inexistent page", func() {
			r, err := c.Get(s.URL + "/nonexistent.i18n.html")
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusNotFound)
		})

		Convey("Get /index.i18n.html from / using default language", func() {
			r, err := c.Get(s.URL + "/")
			So(err, ShouldBeNil)
			content, err := ioutil.ReadAll(r.Body)
			So(string(content), ShouldEqual, ">> trans (00:00:10) -> Root Index")
		})

		Convey("Non index page using default language", func() {
			r, err := c.Get(s.URL + "/some/random/page.i18n.html")
			So(err, ShouldBeNil)
			content, err := ioutil.ReadAll(r.Body)
			So(string(content), ShouldEqual, "== trans (00:00:10) -> Some Random Page")
		})

		Convey("Page cache tests", func() {
			r, err := c.Get(s.URL + "/index.i18n.html")
			So(err, ShouldBeNil)
			content, err := ioutil.ReadAll(r.Body)
			So(string(content), ShouldEqual, ">> trans (00:00:10) -> Root Index")

			Convey("Page is not re-rendered if the file has not changed", func() {
				now.now = now.now.Add(1 * time.Second)
				r, err := c.Get(s.URL + "/index.i18n.html")
				So(err, ShouldBeNil)
				content, err := ioutil.ReadAll(r.Body)
				So(string(content), ShouldEqual, ">> trans (00:00:10) -> Root Index")
			})

			Convey("Page needs to be re-rendered as the source file changes", func() {
				fs.contents["/index.i18n.html"].modificationTime = testutil.MustParseTime(`2000-01-01 00:42:30 +0000`)
				now.now = now.now.Add(1 * time.Second)
				r, err := c.Get(s.URL + "/index.i18n.html")
				So(err, ShouldBeNil)
				content, err := ioutil.ReadAll(r.Body)
				So(string(content), ShouldEqual, ">> trans (00:00:11) -> Root Index")
			})
		})
	})
}
