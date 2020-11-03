package i18n

import (
	"bytes"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/version"
	"golang.org/x/text/language"
	"golang.org/x/text/message/catalog"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"text/template"
	"time"
)

func DefaultWrap(h http.Handler, fs http.FileSystem, catalog catalog.Catalog) *Wrapper {
	return Wrap(h, &FileSystemContents{fs: fs}, translator.New(catalog), &now{})
}

type FileSystemContents struct {
	fs http.FileSystem
}

type file struct {
	http.File
}

func (f *file) ModificationTime() time.Time {
	s, err := f.Stat()
	errorutil.MustSucceed(err)

	return s.ModTime()
}

func (c *FileSystemContents) Reader(path string) (File, error) {
	f, err := c.fs.Open(path)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &file{f}, nil
}

type now struct{}

func (*now) Now() time.Time {
	return time.Now()
}

type File interface {
	io.Reader
	ModificationTime() time.Time
}

type Contents interface {
	Reader(path string) (File, error)
}

type Now interface {
	Now() time.Time
}

type cacheKey struct {
	path string
	tag  language.Tag
	time time.Time
}

type cache struct {
	// NOTE: this cache is not atomic. It's possible that the same page is rendered
	// many times if more than one request is done in between a Load() and a Store() call
	// but this is good enough as no race conditions will exist and the contents will always
	// be the same in production, as the source files don't change over the application lifetime,
	// as they are static data.
	m sync.Map
}

func (c *cache) onKey(key cacheKey, w io.Writer, gen func() []byte) error {
	reply := func(b []byte) error {
		r := bytes.NewReader(b)

		if _, err := io.Copy(w, r); err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}

	if v, ok := c.m.Load(key); ok {
		b := v.([]byte)
		return reply(b)
	}

	newContent := gen()

	c.m.Store(key, newContent)

	return reply(newContent)
}

type Wrapper struct {
	h           http.Handler
	contents    Contents
	translators translator.Translators
	now         Now
	cache       cache
}

func Wrap(h http.Handler, contents Contents, translators translator.Translators, now Now) *Wrapper {
	return &Wrapper{h: h, contents: contents, translators: translators, now: now}
}

func (s *Wrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := func() string {
		trimmed := strings.TrimSpace(r.URL.Path)

		if strings.HasSuffix(trimmed, "/") {
			return trimmed + "index.i18n.html"
		}

		return trimmed
	}()

	if !strings.HasSuffix(path, ".i18n.html") {
		s.h.ServeHTTP(w, r)
		return
	}

	f, err := s.contents.Reader(path)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	lang, _ := r.Cookie("lang")

	accept := r.Header.Get("Accept-Language")

	tag, _ := language.MatchStrings(s.translators.Matcher(), lang.String(), accept)

	err = s.cache.onKey(cacheKey{path: path, time: f.ModificationTime(), tag: tag}, w,
		func() []byte {

			content, err := ioutil.ReadAll(f)

			errorutil.MustSucceed(err)

			translatorByTag := s.translators.Translator(tag, s.now.Now())

			translate := func(s string, args ...interface{}) (string, error) {
				transformed := translator.TransformTranslation(s)
				return translatorByTag.Translate(transformed, args)
			}

			t, err := template.New("root").
				Funcs(template.FuncMap{
					"translate":  translate,
					"appVersion": func() string { return version.Version },
				}).
				Parse(string(content))

			errorutil.MustSucceed(err)

			buffer := bytes.Buffer{}

			err = t.Execute(&buffer, []interface{}{})

			errorutil.MustSucceed(err)

			return buffer.Bytes()
		})

	// TODO: handle this error, as it might be caused by some issue with the connection with the client
	errorutil.MustSucceed(err)
}
