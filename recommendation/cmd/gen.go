// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build ignore

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"net/http"
	"net/url"
	"github.com/rs/zerolog/log"
)

type Link struct {
	Link string `json:"link"`
	ID   string `json:"id"`
}

func main() {
	var (
		mappingFile string
	)

	flag.StringVar(&mappingFile, "mapping-file", "", "filepath to mapping")

	flag.Parse()

	b, err := ioutil.ReadFile(mappingFile)
	if err != nil {
		log.Fatal().Err(err).Msg("could not read file")
	}

	links := make([]Link, 0)
	if err := json.Unmarshal(b, &links); err != nil {
		log.Fatal().Err(err).Msg("could not unmarshal")
	}

	content, err := generateFileContent(links)
	if err != nil {
		log.Fatal().Err(err).Msg("could not unmarshal")
	}

	if err := ioutil.WriteFile("generated_link_list.go", content, 0600); err != nil {
		log.Fatal().Err(err).Msg("could not write to file ")
	}
}

func generateFileContent(links []Link) ([]byte, error) {
	code := []byte(`package recommendation

`)
	if len(links) > 0 {
		code = append(code, []byte(`func init() {`)...)
		code = append(code, generateLinksList(links)...)
		code = append(code, []byte(`}
`)...)
	}

	code, err := format.Source(code)
	if err != nil {
		return nil, err
	}

	return code, err
}

func generateLinksList(links []Link) []byte {
	var code string

	for _, l := range links {
		if _, err := url.Parse(l.Link); err != nil {
			log.Info().Msgf("url is bad: %w", err)
		}

		// nolint:noctx
		resp, err := http.Get(l.Link)
		if err != nil {
			log.Info().Err(err).Msg("Warning url is not reachable")
		} else if resp.StatusCode >= 400 {
			log.Info().Msgf("Warning url is not reachable status code: %d", resp.StatusCode)
		}

		if resp != nil {
			resp.Body.Close()
		}

		code += fmt.Sprintf("\n\t links = append(links, Link{Link: \"%s\", ID: \"%s\"})", l.Link, l.ID)
	}

	return []byte(code)
}
