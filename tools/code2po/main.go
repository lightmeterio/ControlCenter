package main

import (
	"flag"
	"github.com/robfig/gettext-go/gettext/po"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"text/template/parse"
)

var (
	rootDir = flag.String("i", "", "root directory to look for files")
	outfile = flag.String("o", "", "path for po file to write")
	pattern = flag.String("p", `.i18n.html$`, "filename pattern to extract strings")
)

type messages map[string]*po.Message

func visitNode(node parse.Node, messages messages, filename string) {
	if l, ok := node.(*parse.ListNode); ok {
		for _, sn := range l.Nodes {
			visitNode(sn, messages, filename)
		}

		return
	}

	n, ok := node.(*parse.ActionNode)

	if !ok {
		return
	}

	for _, c := range n.Pipe.Cmds {
		if len(c.Args) < 2 {
			continue
		}

		id, ok := c.Args[0].(*parse.IdentifierNode)

		if !ok || id.String() != "translate" {
			continue
		}

		s, ok := c.Args[1].(*parse.StringNode)

		if !ok {
			panic("Wrong Translate call!")
		}

		msg, alreadyExists := messages[s.Text]

		if alreadyExists {
			msg.ReferenceLine = append(msg.ReferenceLine, n.Line)
			msg.ReferenceFile = append(msg.ReferenceFile, filename)
			continue
		}

		messages[s.Text] = &po.Message{
			MsgId:  s.Text,
			MsgStr: s.Text,
			Comment: po.Comment{
				ReferenceLine: []int{n.Line},
				ReferenceFile: []string{filename},
				StartLine:     n.Line,
			},
		}
	}
}

func buildTemplateFromFile(filename string) *template.Template {
	f, err := os.Open(filename)

	if err != nil {
		panic(err)
	}

	content, err := ioutil.ReadAll(f)

	if err != nil {
		panic(err)
	}

	t, err := template.New("root").
		Funcs(template.FuncMap{"translate": func(s string, args ...interface{}) string {
			return s
		}}).
		Parse(string(content))

	if err != nil {
		panic(err)
	}

	return t
}

func main() {
	// TODO: implement updating po file with new translations,
	// not to lose the already extracted ones
	flag.Parse()

	f := po.File{}

	f.MimeHeader.Language = "en"

	r := regexp.MustCompile(*pattern)

	messages := messages{}

	err := filepath.Walk(*rootDir, func(p string, info os.FileInfo, err error) error {
		if !r.Match([]byte(p)) {
			return nil
		}

		filename := p

		log.Println("Extracting strings from filename", filename)

		t := buildTemplateFromFile(filename)

		visitNode(t.Root, messages, filename)

		return nil
	})

	if err != nil {
		panic(err)
	}

	for _, m := range messages {
		f.Messages = append(f.Messages, *m)
	}

	sort.Slice(f.Messages, func(i, j int) bool {
		return strings.Compare(f.Messages[i].MsgId, f.Messages[j].MsgId) < 0
	})

	err = f.Save(*outfile)

	if err != nil {
		panic(err)
	}
}
