// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"flag"
	"fmt"
	"github.com/chai2010/gettext-go/po"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/tools/poutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var langFileRegexp = regexp.MustCompile(`.*\/([^/]+)\/LC_MESSAGES\/`)

var messages = map[string]po.Message{}

func main() {
	var (
		rootDir       = flag.String("i", "", "root directory to look for files")
		outfile       = flag.String("o", "", "path for po file to write")
		debugMode     = flag.Bool("debugMode", false, "debug mode")
		addMissingIDs = flag.Bool("ids", false, "add missing ids")
	)

	flag.Parse()

	log.Info().Msg("parse all files in dir")

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if *debugMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	fset := token.NewFileSet()

	pkgs, err := ParseAllDir(fset, *rootDir, func(os.FileInfo) bool { return true }, parser.ParseComments, *debugMode)
	if err != nil {
		log.Panic().Err(err).Msg("could not parse dir")
	}

	log.Info().Msgf("files count: %v", len(pkgs))
	log.Info().Msgf("iterate over all files and extract all language keys")

	for _, v := range pkgs {
		for _, vv := range v.Files {
			log.Debug().Msgf("file: %v", vv.Name)
			ast.Walk(VisitorFunc(FindLangaugeKeys(*debugMode, fset)), vv)
		}
	}

	messagesList := make([]po.Message, 0)

	for _, message := range messages {
		messagesList = append(messagesList, message)
	}

	if *addMissingIDs {
		err := poutil.SaveDifference(*outfile, messagesList)
		if err != nil {
			panic(err)
		}

		return
	}

	f := po.File{}

	if strings.HasSuffix(*outfile, ".po") {
		r := langFileRegexp.FindSubmatch([]byte(*outfile))

		if len(r) != 2 {
			panic("Invalid output file:" + *outfile)
		}

		// Write the language
		f.MimeHeader.Language = string(r[1])
	}

	// use custom save and pre process
	err = poutil.Save(*outfile, poutil.Data(messagesList, f.MimeHeader.String()))
	if err != nil {
		panic(err)
	}
}

type VisitorFunc func(n ast.Node) ast.Visitor

func (f VisitorFunc) Visit(n ast.Node) ast.Visitor {
	return f(n)
}

const FuncI18n = "I18n"

// nolint:gocriticm,nestif
func FindLangaugeKeys(debugMode bool, fset *token.FileSet) func(n ast.Node) ast.Visitor {
	return func(n ast.Node) ast.Visitor {
		log.Debug().Msgf("")
		log.Debug().Msgf("ast node:")
		log.Debug().Msgf("verbose value: %#v", n)
		log.Debug().Msgf("type: %T", n)
		log.Debug().Msgf("value: %v", n)

		switch n := n.(type) {
		case *ast.Package:
			return VisitorFunc(FindLangaugeKeys(debugMode, fset))
		case *ast.File:
			return VisitorFunc(FindLangaugeKeys(debugMode, fset))
		case *ast.GenDecl:
			if n.Tok == token.TYPE {
				return VisitorFunc(FindLangaugeKeys(debugMode, fset))
			}
		case *ast.FuncDecl:
			return VisitorFunc(FindLangaugeKeys(debugMode, fset))
		case *ast.ReturnStmt:
			return VisitorFunc(FindLangaugeKeys(debugMode, fset))
		case *ast.BlockStmt:
			return VisitorFunc(FindLangaugeKeys(debugMode, fset))
		case *ast.ExprStmt:
			return VisitorFunc(FindLangaugeKeys(debugMode, fset))
		case ast.Stmt:
			return VisitorFunc(FindLangaugeKeys(debugMode, fset))
		case *ast.CallExpr:

			filename := fset.File(n.Pos()).Name()
			line := fset.File(n.Pos()).Line(n.Pos())
			log.Debug().Msgf("file: %s line: %d", filename, line)

			if _, ok := n.Fun.(*ast.SelectorExpr); ok {
				if n.Fun.(*ast.SelectorExpr).Sel.Name == FuncI18n {
					MustStoreID(n.Args[0], filename, fset.File(n.Pos()).Line(n.Pos()))
					return nil
				}
			}

			ident, ok := n.Fun.(*ast.Ident)
			if !ok {
				return nil
			}

			if ident.Obj == nil {
				return nil
			}

			if assignStmt, ok := ident.Obj.Decl.(*ast.AssignStmt); ok {
				if ident, ok := assignStmt.Rhs[0].(*ast.Ident); ok {
					if funcDecl, ok := ident.Obj.Decl.(*ast.FuncDecl); ok {
						if funcDecl.Name.Name == FuncI18n {
							MustStoreID(n.Args[0], filename, fset.File(n.Pos()).Line(n.Pos()))
						}
					}
				}
			} else if ident.Name == FuncI18n {
				MustStoreID(n.Args[0], filename, fset.File(n.Pos()).Line(n.Pos()))
			}
		}

		return nil
	}
}

func MustStoreID(expr ast.Expr, filename string, line int) {
	err := StoreMsgID(expr, filename, line)
	if err != nil {
		errorutil.LogErrorf(err, "the expression is bad")
		os.Exit(1)
	}
}

func StoreMsgID(e ast.Expr, filename string, line int) error {
	if typ, ok := e.(*ast.BasicLit); ok {
		if typ.Kind == 9 {
			cleanValue := strings.TrimFunc(typ.Value, func(r rune) bool {
				return r == '"' || r == '`'
			})

			log.Info().Msgf("MsgId: %s", cleanValue)

			message := po.Message{
				MsgId:  cleanValue,
				MsgStr: cleanValue,
				Comment: po.Comment{
					ReferenceLine: []int{line},
					ReferenceFile: []string{filename},
					StartLine:     line,
				},
			}

			messages[cleanValue] = message
		}
	} else {
		return errorutil.Wrap(fmt.Errorf("Error custom types and variables are not allowed in combination with I18n: %v", e))
	}

	return nil
}

// nolint:gocriticm,nestif
func ParseAllDir(fset *token.FileSet, path string, filter func(os.FileInfo) bool, mode parser.Mode, debugMode bool) (map[string]*ast.Package, error) {
	pkgs := make(map[string]*ast.Package)

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {

		if strings.Contains(path, "node_modules") || strings.Contains(path, "vendor") || strings.Contains(path, ".git") || (!debugMode && strings.Contains(path, "gotestdata")) {
			return nil
		}

		if strings.HasSuffix(info.Name(), ".go") && (filter == nil || filter(info)) {

			log.Debug().Msgf("file: %s", filepath.Join(path, info.Name()))

			if src, err := parser.ParseFile(fset, path, nil, mode); err == nil {
				name := src.Name.Name
				pkg, found := pkgs[name]
				if !found {
					pkg = &ast.Package{
						Name:  name,
						Files: make(map[string]*ast.File),
					}
					pkgs[name] = pkg
				}
				pkg.Files[path] = src
			} else {
				return errorutil.Wrap(err)
			}
		} else {
			if info.IsDir() {
				log.Debug().Msgf("dir: %s", path)
			}
		}
		return nil
	})

	return pkgs, err
}
