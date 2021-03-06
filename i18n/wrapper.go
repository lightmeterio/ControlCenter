// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package i18n

import (
	"fmt"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/po"
	"gitlab.com/lightmeter/controlcenter/util/httputil"

	"golang.org/x/text/message/catalog"
	"net/http"
)

func NewService(catalog catalog.Catalog) *Service {
	return &Service{
		translators: translator.New(catalog),
	}
}

type Service struct {
	translators translator.Translators
}

type LanguagePair struct {
	Value string `json:"value"`
	Key   string `json:"key"`
}

// order of values matters
var languages = []LanguagePair{
	{Key: "English", Value: po.English}, // default value
	{Key: "Deutsch", Value: po.German},
	{Key: "Português do Brasil", Value: po.BrazilianPortuguese},
}

func GetLanguages() []LanguagePair {
	return languages
}

func (s *Service) LanguageMetaDataHandler(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return httperror.NewHTTPStatusCodeError(http.StatusMethodNotAllowed, fmt.Errorf("Error http method mismatch: %v", r.Method))
	}

	type MetaData struct {
		Languages []LanguagePair `json:"languages"`
	}

	copyLanguages := GetLanguages()

	m := MetaData{
		Languages: copyLanguages,
	}

	return httputil.WriteJson(w, &m, http.StatusOK)
}
