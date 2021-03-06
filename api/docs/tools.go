// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build tools
// +build tools

package docs

// added to prevent `go mod tidy` of removing it as a dependency
// of the docs.go file generated by swag
import _ "github.com/alecthomas/template"
