package main

import (
	"flag"
	"fmt"
	"github.com/robfig/gettext-go/gettext/po"
	"gitlab.com/lightmeter/controlcenter/tools/poutil"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var messages sync.Map

func main() {
	var (
		rootDir   = flag.String("i", "", "root directory to look for files")
		outfile   = flag.String("o", "", "path for po file to write")
		debugMode = flag.Bool("debugMode", false, "debug mode")
	)

	flag.Parse()

	log.Println("parse all files in dir")

	fset := token.NewFileSet()

	pkgs, err := ParseAllDir(fset, *rootDir, func(os.FileInfo) bool { return true }, parser.ParseComments)
	if err != nil {
		log.Panicln(err)
	}

	log.Println("files count: ", len(pkgs))
	log.Println("iterate over all files and extract all language keys")

	for _, v := range pkgs {
		for _, vv := range v.Files {
			log.Println("file: ", vv.Name)
			ast.Walk(VisitorFunc(FindLangaugeKeys(*debugMode, fset)), vv)
		}
	}

	f := po.File{}

	messages.Range(func(key, value interface{}) bool {
		message := value.(po.Message)
		f.Messages = append(f.Messages, message)
		return true
	})

	// use custom save and pre process
	err = poutil.Save(*outfile, poutil.Data(f.Messages, f.MimeHeader.String()))
	if err != nil {
		panic(err)
	}
}

type VisitorFunc func(n ast.Node) ast.Visitor

func (f VisitorFunc) Visit(n ast.Node) ast.Visitor {
	return f(n)
}

const FuncI18n = "I18n"

func FindLangaugeKeys(debugNode bool, fset *token.FileSet) func(n ast.Node) ast.Visitor {
	return func(n ast.Node) ast.Visitor {
		if debugNode {
			log.Println("")
			log.Println("ast node:")
			log.Println(fmt.Sprintf("verbose value: %#v", n))
			log.Println(fmt.Sprintf("type: %T", n))
			log.Println(fmt.Sprintf("value: %v", n))
		}

		switch n := n.(type) {
		case *ast.Package:
			return VisitorFunc(FindLangaugeKeys(debugNode, fset))
		case *ast.File:
			return VisitorFunc(FindLangaugeKeys(debugNode, fset))
		case *ast.GenDecl:
			if n.Tok == token.TYPE {
				return VisitorFunc(FindLangaugeKeys(debugNode, fset))
			}
		case *ast.FuncDecl:
			return VisitorFunc(FindLangaugeKeys(debugNode, fset))
		case *ast.ReturnStmt:
			return VisitorFunc(FindLangaugeKeys(debugNode, fset))
		case *ast.BlockStmt:
			return VisitorFunc(FindLangaugeKeys(debugNode, fset))
		case *ast.ExprStmt:
			return VisitorFunc(FindLangaugeKeys(debugNode, fset))
		case ast.Stmt:
			return VisitorFunc(FindLangaugeKeys(debugNode, fset))
		case *ast.CallExpr:
			if _, ok := n.Fun.(*ast.SelectorExpr); ok {
				if n.Fun.(*ast.SelectorExpr).Sel.Name == FuncI18n {
					StoreMsgID(n.Args[0])
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

			name := fset.File(n.Pos()).Name()
			log.Println("file:", name, " line:", fset.File(n.Pos()).Line(n.Pos()))

			if assignStmt, ok := ident.Obj.Decl.(*ast.AssignStmt); ok {
				if ident, ok := assignStmt.Rhs[0].(*ast.Ident); ok {
					if funcDecl, ok := ident.Obj.Decl.(*ast.FuncDecl); ok {
						if funcDecl.Name.Name == FuncI18n {
							StoreMsgID(n.Args[0])
						}
					}
				}
			} else if ident.Name == FuncI18n {
				StoreMsgID(n.Args[0])
			}
		}

		return nil
	}
}

func StoreMsgID(e ast.Expr) {
	if typ, ok := e.(*ast.BasicLit); ok {
		if typ.Kind == 9 {
			cleanValue := strings.TrimFunc(typ.Value, func(r rune) bool {
				return r == '"'
			})

			log.Println("MsgId: ", cleanValue)

			message := po.Message{
				MsgId:  cleanValue,
				MsgStr: cleanValue,
			}

			messages.Store(cleanValue, message)
		}
	} else {
		log.Println("Error custom types and variables are not allowed in combination with I18n: ", e)
	}
}

func ParseAllDir(fset *token.FileSet, path string, filter func(os.FileInfo) bool, mode parser.Mode) (map[string]*ast.Package, error) {
	pkgs := make(map[string]*ast.Package)

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {

		if strings.Contains(path, "vendor") || strings.Contains(path, ".git") || strings.Contains(path, "gotestdata")  {
			return nil
		}

		if strings.HasSuffix(info.Name(), ".go") && (filter == nil || filter(info)) {
			fmt.Println("file: ", filepath.Join(path, info.Name()))
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
				log.Println("Error ", err)
			}
		} else {
			if info.IsDir() {
				fmt.Println("dir: ", path)
			}
		}
		return nil
	})

	return pkgs, err
}
