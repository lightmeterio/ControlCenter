package main

import (
	"flag"
	"fmt"
	"github.com/robfig/gettext-go/gettext/po"
	"gitlab.com/lightmeter/controlcenter/tools/poutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var messages = map[string]po.Message{}

func main() {
	var (
		rootDir       = flag.String("i", "", "root directory to look for files")
		outfile       = flag.String("o", "", "path for po file to write")
		debugMode     = flag.Bool("debugMode", false, "debug mode")
		addMissingIDs = flag.Bool("ids", false, "add missing ids")
	)

	flag.Parse()

	log.Println("parse all files in dir")

	fset := token.NewFileSet()

	pkgs, err := ParseAllDir(fset, *rootDir, func(os.FileInfo) bool { return true }, parser.ParseComments, *debugMode)
	if err != nil {
		log.Panicln(err)
	}

	log.Println("files count: ", len(pkgs))
	log.Println("iterate over all files and extract all language keys")

	for _, v := range pkgs {
		for _, vv := range v.Files {
			if *debugMode {
				log.Println("file: ", vv.Name)
			}

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
		if debugMode {
			log.Println("")
			log.Println("ast node:")
			log.Println(fmt.Sprintf("verbose value: %#v", n))
			log.Println(fmt.Sprintf("type: %T", n))
			log.Println(fmt.Sprintf("value: %v", n))
		}

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
			if debugMode {
				log.Println("file:", filename, " line:", line)
			}

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
		log.Println(err)
		os.Exit(1)
	}
}

func StoreMsgID(e ast.Expr, filename string, line int) error {
	if typ, ok := e.(*ast.BasicLit); ok {
		if typ.Kind == 9 {
			cleanValue := strings.TrimFunc(typ.Value, func(r rune) bool {
				return r == '"'
			})

			log.Println("MsgId: ", cleanValue)

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

			if debugMode {
				fmt.Println("file: ", filepath.Join(path, info.Name()))
			}

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
			if debugMode && info.IsDir() {
				fmt.Println("dir: ", path)
			}
		}
		return nil
	})

	return pkgs, err
}
