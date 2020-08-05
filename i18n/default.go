package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
	"io"
	"net/http"
)

type translators struct {
	catalog catalog.Catalog
}

func newTranslators(catalog catalog.Catalog) Translators {
	return &translators{catalog: catalog}
}

type translator struct {
	printer *message.Printer
}

func newTranslator(tag language.Tag, c catalog.Catalog) *translator {
	return &translator{printer: message.NewPrinter(tag, message.Catalog(c))}
}

func (t *translator) Translate(s string, args ...interface{}) (string, error) {
	return t.printer.Sprintf(message.Key(s, s), args), nil
}

func (t *translators) Translator(tag language.Tag) Translator {
	return newTranslator(tag, t.catalog)
}

func (t *translators) Matcher() language.Matcher {
	return t.catalog.Matcher()
}

func DefaultWrap(h http.Handler, fs http.FileSystem, catalog catalog.Catalog) *Wrapper {
	return Wrap(h, &FileSystemContents{fs: fs}, newTranslators(catalog))
}

type FileSystemContents struct {
	fs http.FileSystem
}

func (c *FileSystemContents) Reader(path string) (io.Reader, error) {
	return c.fs.Open(path)
}
