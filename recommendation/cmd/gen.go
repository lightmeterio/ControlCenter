package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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
		log.Fatal(err)
	}

	links := make([]Link, 0)
	if err := json.Unmarshal(b, &links); err != nil {
		log.Fatal(err)
	}

	content, err := generateFileContent(links)
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile("generated_link_list.go", content, 0600); err != nil {
		log.Fatal(err)
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
			log.Panicln("url is bad: ", err)
		}

		// nolint:noctx
		resp, err := http.Get(l.Link)
		if err != nil {
			log.Println("Warning url is not reachable ", err)
		} else if resp.StatusCode >= 400 {
			log.Println("Warning url is not reachable status code: ", resp.StatusCode)
		}

		if resp != nil {
			resp.Body.Close()
		}

		code += fmt.Sprintf("\n\t links = append(links, Link{Link: \"%s\", ID: \"%s\"})", l.Link, l.ID)
	}

	return []byte(code)
}
