// +build ignore

package main

import (
	"github.com/shurcooL/vfsgen"
	"log"
	"net/http"
)

func main() {
	err := vfsgen.Generate(
		http.Dir("./static"),
		vfsgen.Options{
			Filename:     "./http_vfsdata.go",
			PackageName:  "main",
			VariableName: "httpAssets",
		})

	if err != nil {
		log.Fatalln(err)
	}
}
