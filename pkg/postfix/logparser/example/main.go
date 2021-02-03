// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * read log lines from stdin and pring the parsed result as JSON objects
 */

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		if !scanner.Scan() {
			return
		}

		h, p, err := parser.Parse(scanner.Bytes())

		if !parser.IsRecoverableError(err) {
			continue
		}

		name := func() string {
			t := reflect.TypeOf(p)

			if t == nil {
				return "none"
			}

			return t.Name()
		}()

		if j, err := json.Marshal([]interface{}{h, name, p}); err == nil {
			fmt.Println(string(j))
		}
	}
}
