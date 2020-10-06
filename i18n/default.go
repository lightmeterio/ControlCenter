package i18n

import (
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
	"net/http"
	"time"
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

func newTranslator(tag language.Tag, c catalog.Catalog, accessTime time.Time) *translator {
	return &translator{printer: message.NewPrinter(tag, message.Catalog(c))}
}

func (t *translator) Translate(s string, args ...interface{}) (string, error) {
	return t.printer.Sprintf(message.Key(s, s), args), nil
}

func (t *translators) Translator(tag language.Tag, accessTime time.Time) Translator {
	return newTranslator(tag, t.catalog, accessTime)
}

func (t *translators) Matcher() language.Matcher {
	return t.catalog.Matcher()
}

type now struct {
}

func (*now) Now() time.Time {
	return time.Now()
}

func DefaultWrap(h http.Handler, fs http.FileSystem, catalog catalog.Catalog) *Wrapper {
	return Wrap(h, &FileSystemContents{fs: fs}, newTranslators(catalog), &now{})
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
