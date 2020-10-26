package poutil

import (
	"bytes"
	"fmt"
	"github.com/robfig/gettext-go/gettext/po"
	"io/ioutil"
	"sort"
	"strings"
)

// Save returns a po file format data.
func Data(messages []po.Message, mimeHeader string) []byte {
	sort.Slice(messages, func(i, j int) bool {
		si := messages[i].MsgId
		sj := messages[j].MsgId
		siLower := strings.ToLower(si)
		sjLower := strings.ToLower(sj)
		if siLower == sjLower {
			return si < sj
		}
		return siLower < sjLower
	})

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "%s\n", mimeHeader)

	for i := 0; i < len(messages); i++ {
		fmt.Fprintf(&buf, "%s\n", messages[i].String())
	}

	return buf.Bytes()
}

// Save saves a po file.
func Save(name string, buff []byte) error {
	return ioutil.WriteFile(name, buff, 0600)
}

