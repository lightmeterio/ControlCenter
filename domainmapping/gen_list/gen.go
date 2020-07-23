// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/util"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	f, err := os.Open("mapping.json")
	util.MustSucceed(err, "")
	b, err := ioutil.ReadAll(f)
	util.MustSucceed(err, "")
	l := domainmapping.RawList{}
	util.MustSucceed(json.Unmarshal(b, &l), "Invalid domain mapping file")
	_, err = domainmapping.Mapping(l)
	util.MustSucceed(err, "Malformed domain mapping file")
	_, err = json.Marshal(l)
	util.MustSucceed(err, "")
	util.MustSucceed(ioutil.WriteFile("generated_list.go", fileContent(l), 0600), "")
}

func fileContent(l domainmapping.RawList) []byte {
	return []byte(`package domainmapping

import "gitlab.com/lightmeter/controlcenter/util"

func init() {
	DefaultMapping = mustBeValidList()
}

func mustBeValidList() *Mapper {
	m, err := Mapping(mustParse())
	util.MustSucceed(err, "Invalid Domain List")
	return &m
}

func mustParse() RawList {
	l := RawList{} ` + genList(l) + `
	return l
}
`)
}

func genList(l domainmapping.RawList) string {
	s := ""

	for k, v := range l {
		s += fmt.Sprintf("\n\tl[`%s`] = make([]string, 0, %d)", strings.ToLower(k), len(v))
	}

	for k, v := range l {
		for _, d := range v {
			s += fmt.Sprintf("\n\tl[`%s`] = append(l[`%s`], `%s`)", strings.ToLower(k), strings.ToLower(k), strings.ToLower(d))
		}
	}

	return s
}