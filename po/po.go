//go:generate go run ../tools/po2go/main.go -i . -o generated.go

package po

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message/catalog"
)

var (
	FallbackLanguage = language.English

	// This is the entrypoint for the generated dictionaries during build time
	DefaultCatalog = catalog.NewBuilder(catalog.Fallback(FallbackLanguage))
)
