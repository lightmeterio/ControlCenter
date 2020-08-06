package i18n

import (
	"bytes"
	"gitlab.com/lightmeter/controlcenter/util"
	"golang.org/x/text/language"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"text/template"
	"time"
)

type File interface {
	io.Reader
	ModificationTime() time.Time
}

type Contents interface {
	Reader(path string) (File, error)
}

type Translators interface {
	Translator(language.Tag, time.Time) Translator
	Matcher() language.Matcher
}

type Translator interface {
	Translate(string, ...interface{}) (string, error)
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
	m sync.Map
}

func (c *cache) onKey(key cacheKey, w io.Writer, gen func() []byte) error {
	// NOTE: this cache is not atomic. It's possible that the same page is rendered
	// many times if more than one request is done in between a Load() and a Store() call
	// but this is good enough as no race conditions will exist and the contents will always
	// be the same in production, as the source files don't change over the application lifetime,
	// as they are static data.

	reply := func(b []byte) error {
		r := bytes.NewReader(b)

		if _, err := io.Copy(w, r); err != nil {
			return util.WrapError(err)
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
	translators Translators
	now         Now
	cache       cache
}

func Wrap(h http.Handler, contents Contents, translators Translators, now Now) *Wrapper {
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
			translator := s.translators.Translator(tag, s.now.Now())

			content, err := ioutil.ReadAll(f)

			util.MustSucceed(err, "")

			t, err := template.New("root").
				Funcs(template.FuncMap{"translate": translator.Translate}).
				Parse(string(content))

			util.MustSucceed(err, "")

			buffer := bytes.Buffer{}

			err = t.Execute(&buffer, []interface{}{})

			util.MustSucceed(err, "")

			return buffer.Bytes()
		})

	// TODO: handle this error, as it might be caused by some issue with the connection with the client
	util.MustSucceed(err, "")
}
