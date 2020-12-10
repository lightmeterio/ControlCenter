package main

import (
	"github.com/shurcooL/vfsgen"
	"log"
	"net/http"
)

func main() {
	err := vfsgen.Generate(
		http.Dir("../www"),
		vfsgen.Options{
			Filename:     "http_vfsdata.go",
			PackageName:  "staticdata",
			BuildTags:    "include",
			VariableName: "HttpAssets",
		})

	if err != nil {
		log.Fatalln(err)
	}
}
