// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package poutil

import (
	"bytes"
	"fmt"
	"github.com/chai2010/gettext-go/po"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io/ioutil"
	"os"
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
// nolint:gocriticm,nestif
func SaveDifference(name string, messagesList []po.Message) error {
	var newFile *po.File

	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			newFile = &po.File{}
		}
	} else {
		content, err := ioutil.ReadFile(name)
		if err != nil {
			return errorutil.Wrap(err)
		}

		newFile, err = po.Load(content)
		if err != nil {
			return errorutil.Wrap(err)
		}
	}

	messageMap := make(map[string]po.Message)
	for _, message := range newFile.Messages {
		messageMap[message.MsgId] = message
	}

	for _, message := range messagesList {
		// Skip all messages which are available in messages to avoid generation of duplicates
		if oldMessage, ok := messageMap[message.MsgId]; ok {
			if oldMessage.StartLine != message.StartLine {
				oldMessage.StartLine = message.StartLine
			}

			oldFilename := getReferenceFileFirst(oldMessage.Comment.ReferenceFile)
			newFilename := getReferenceFileFirst(message.Comment.ReferenceFile)

			if oldFilename != newFilename {
				comment := oldMessage.Comment

				if comment.ReferenceFile == nil {
					comment.ReferenceFile = make([]string, 1)
				}

				comment.ReferenceFile[0] = newFilename
				oldMessage.Comment = comment
			}

			oldLine := getReferenceLineFirst(oldMessage.Comment.ReferenceLine)
			newLine := getReferenceLineFirst(message.Comment.ReferenceLine)

			if newLine != oldLine {
				comment := oldMessage.Comment
				if comment.ReferenceLine == nil {
					comment.ReferenceLine = make([]int, 1)
				}

				comment.ReferenceLine[0] = newLine
				oldMessage.Comment = comment
			}

			messageMap[message.MsgId] = oldMessage

			continue
		}

		message.MsgStr = " "
		message.Flags = append(message.Flags, "Fuzzy")
		messageMap[message.MsgId] = message
	}

	messagesList = make([]po.Message, 0)
	for _, message := range messageMap {
		messagesList = append(messagesList, message)
	}

	// use custom save and pre process
	err := Save(name, Data(messagesList, newFile.MimeHeader.String()))
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func getReferenceLineFirst(line []int) int {
	if len(line) > 0 {
		return line[0]
	}

	return -1
}

func getReferenceFileFirst(file []string) string {
	if len(file) > 0 {
		return file[0]
	}

	return ""
}
