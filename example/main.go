/**
 * read log lines from stdin and pring the parsed result as JSON objects
 */

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		if !scanner.Scan() {
			return
		}

		h, p, err := parser.Parse(scanner.Bytes())

		if err != nil && err == rawparser.InvalidHeaderLineError {
			continue
		}

		if j, err := json.Marshal([]interface{}{h, p}); err == nil {
			fmt.Println(string(j))
		}
	}

}
