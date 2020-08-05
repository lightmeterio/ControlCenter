package i18n

import (
	"gitlab.com/lightmeter/controlcenter/util"
	"golang.org/x/text/language"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"
)

type Contents interface {
	Reader(path string) (io.Reader, error)
}

type Translators interface {
	Translator(language.Tag) Translator
	Matcher() language.Matcher
}

type Translator interface {
	Translate(string, ...interface{}) (string, error)
}

type Wrapper struct {
	h           http.Handler
	contents    Contents
	translators Translators
}

func Wrap(h http.Handler, contents Contents, translators Translators) *Wrapper {
	return &Wrapper{h: h, contents: contents, translators: translators}
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

	// TODO: cache pre-rendered pages

	lang, _ := r.Cookie("lang")

	accept := r.Header.Get("Accept-Language")

	tag, _ := language.MatchStrings(s.translators.Matcher(), lang.String(), accept)

	translator := s.translators.Translator(tag)

	f, err := s.contents.Reader(path)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	content, err := ioutil.ReadAll(f)

	util.MustSucceed(err, "")

	t, err := template.New("root").
		Funcs(template.FuncMap{"translate": translator.Translate}).
		Parse(string(content))

	util.MustSucceed(err, "")

	err = t.Execute(w, []interface{}{})

	util.MustSucceed(err, "")
}
