package translator

import (
	"fmt"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
	"strings"
	"time"
)

type TranslatableStringer interface {
	TplString() string
	Args() []interface{}
}

func Stringfy(s TranslatableStringer) string {
	return fmt.Sprintf(strings.ReplaceAll(s.TplString(), "%%", "%"), s.Args()...)
}

type Translator interface {
	Translate(string, ...interface{}) (string, error)
}

type Translators interface {
	Translator(language.Tag, time.Time) Translator
	Matcher() language.Matcher
}

type translators struct {
	catalog catalog.Catalog
}

func (t *translators) Translator(tag language.Tag, accessTime time.Time) Translator {
	return newTranslator(tag, t.catalog, accessTime)
}

func (t *translators) Matcher() language.Matcher {
	return t.catalog.Matcher()
}

func New(catalog catalog.Catalog) Translators {
	return &translators{catalog: catalog}
}

type translator struct {
	printer *message.Printer
}

func newTranslator(tag language.Tag, c catalog.Catalog, accessTime time.Time) *translator {
	return &translator{printer: message.NewPrinter(tag, message.Catalog(c))}
}

func (t *translator) Translate(s string, args ...interface{}) (string, error) {
	return t.printer.Sprintf(message.Key(s, s), args), nil
}
