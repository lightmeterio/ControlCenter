// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"github.com/shurcooL/vfsgen"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
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
		errorutil.LogFatalf(err, "Failed to generate static asset file")
	}
}
