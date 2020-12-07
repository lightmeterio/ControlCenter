package i18n

import (
	"fmt"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/po"
	"gitlab.com/lightmeter/controlcenter/util/httputil"

	"golang.org/x/text/message/catalog"
	"net/http"
)

func NewService(catalog catalog.Catalog) *Service {
	return &Service{
		translators:          translator.New(catalog),
	}
}

type Service struct {
	translators          translator.Translators
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
		return httpmiddleware.NewHTTPStatusCodeError(http.StatusMethodNotAllowed, fmt.Errorf("Error http method mismatch: %v", r.Method))
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
