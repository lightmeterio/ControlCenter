package poutil

import (
	"bytes"
	"fmt"
	"github.com/robfig/gettext-go/gettext/po"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
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

// Save difference returns a po file format data.
func SaveDifference(name string, messagesList []po.Message) error {
	content, err := ioutil.ReadFile(name)
	if err != nil {
		return errorutil.Wrap(err)
	}

	newFile, err := po.LoadData(content)
	if err != nil {
		return errorutil.Wrap(err)
	}

	ids := make(map[string]bool)
	for _, message := range newFile.Messages {
		ids[message.MsgId] = true
	}

	for _, message := range messagesList {
		// Skip all messages which are available in messages to avoid generation of duplicates
		if ids[message.MsgId] {
			continue
		}
		newFile.Messages = append(newFile.Messages, message)
	}

	// use custom save and pre process
	err = Save(name, Data(newFile.Messages, newFile.MimeHeader.String()))
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

