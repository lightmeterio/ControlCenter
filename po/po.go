// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

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

const (
	German              string = "de"
	English             string = "en"
	BrazilianPortuguese string = "pt_BR"
)

func IsLanguageSupported(code string) bool {
	switch code {
	case German, English, BrazilianPortuguese:
		return true
	}

	return false
}
