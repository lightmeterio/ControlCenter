// +build ignore

package main

import (
	"github.com/shurcooL/vfsgen"
	"log"
	"net/http"
)

func main() {
	err := vfsgen.Generate(
		http.Dir("./www"),
		vfsgen.Options{
			Filename:     "./staticdata/http_vfsdata.go",
			PackageName:  "staticdata",
			BuildTags:    "release",
			VariableName: "HttpAssets",
		})

	if err != nil {
		log.Fatalln(err)
	}
}
