/**
 * read log lines from stdin and pring the parsed result as JSON objects
 */

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	parser "gitlab.com/lightmeter/postfix-log-parser"
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

		if j, err := json.Marshal([]interface{}{h, p}); err == nil {
			fmt.Println(string(j))
		}
	}

}
