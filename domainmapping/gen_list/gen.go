// +build ignore

package main

import (
	"encoding/json"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/util"
	"io/ioutil"
	"os"
)

func main() {
	f, err := os.Open("list.json")
	util.MustSucceed(err, "")
	b, err := ioutil.ReadAll(f)
	util.MustSucceed(err, "")
	l := domainmapping.RawList{}
	util.MustSucceed(json.Unmarshal(b, &l), "Invalid domain mapping file")
	s, err := json.Marshal(l)
	util.MustSucceed(err, "")
	util.MustSucceed(ioutil.WriteFile("generated_list.go", fileContent(s), 0600), "")
}

func fileContent(s []byte) []byte {
	return []byte(`
package domainmapping

import "encoding/json"
import "gitlab.com/lightmeter/controlcenter/util"

var DefaultMapping = Mapping(mustParse())

func mustParse() RawList {
	l := RawList{}
	util.MustSucceed(json.Unmarshal([]byte(` + "`" + string(s) + "`" + `), &l), "")
	return l
}
`)
}
