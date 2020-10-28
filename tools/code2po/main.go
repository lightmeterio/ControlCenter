package main

import (
	"flag"
	"github.com/robfig/gettext-go/gettext/po"
	"gitlab.com/lightmeter/controlcenter/tools/poutil"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"text/template"
	"text/template/parse"
)

var (
	rootDir       = flag.String("i", "", "root directory to look for files")
	outfile       = flag.String("o", "", "path for po file to write")
	pattern       = flag.String("p", `.i18n.html$`, "filename pattern to extract strings")
	isTemplate    = flag.Bool("pot", false, "use if generating a pot file")
	addMissingIDs = flag.Bool("ids", false, "add missing ids")
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

		if msg, alreadyExists := messages[s.Text]; alreadyExists {
			msg.ReferenceLine = append(msg.ReferenceLine, n.Line)
			msg.ReferenceFile = append(msg.ReferenceFile, filename)

			continue
		}

		translatedMsg := func() string {
			if *isTemplate {
				// a empty string will make the `po` package omit it to the final po file
				// so a one space will be the workaround we'll use :-(
				return " "
			}

			return s.Text
		}

		flags := func() []string {
			if *isTemplate {
				return []string{"fuzzy"}
			}

			return []string{}
		}

		messages[s.Text] = &po.Message{
			MsgId:  s.Text,
			MsgStr: translatedMsg(),
			Comment: po.Comment{
				ReferenceLine: []int{n.Line},
				ReferenceFile: []string{filename},
				StartLine:     n.Line,
				Flags:         flags(),
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
		Funcs(template.FuncMap{
			"translate": func(s string, args ...interface{}) string {
				return s
			},
			"appVersion": func() string {
				return ""
			},
		}).
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

	messagesList := make([]po.Message, 0)
	for _, m := range messages {
		messagesList = append(messagesList, *m)
	}

	if *addMissingIDs {
		err := poutil.SaveDifference(*outfile, messagesList)
		if err != nil {
			panic(err)
		}

		return
	}

	f := po.File{}

	if !*isTemplate {
		f.MimeHeader.Language = "en"
	}

	// use custom save and pre process
	err = poutil.Save(*outfile, poutil.Data(messagesList, f.MimeHeader.String()))
	if err != nil {
		panic(err)
	}
}
