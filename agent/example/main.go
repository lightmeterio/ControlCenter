// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"bytes"
	"context"
	"gitlab.com/lightmeter/controlcenter/agent/driver"
	"gitlab.com/lightmeter/controlcenter/agent/parser"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	d, err := driver.NewDockerDriver(os.Getenv("POSTFIX_CONTAINER"), "0")
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	{
		stdout := bytes.Buffer{}

		if err := d.ExecuteCommand(ctx, []string{"postconf"}, nil, &stdout, io.Discard); err != nil {
			panic(err)
		}

		conf, err := parser.Parse(stdout.Bytes())
		if err != nil {
			panic(err)
		}

		version, err := conf.Resolve("mail_version")
		if err != nil {
			panic(err)
		}

		log.Println("Version: ", version)
	}

	{
		stdout := bytes.Buffer{}

		// send some input to the command. In this case, just invert the content sent
		if err := d.ExecuteCommand(ctx, []string{"rev"}, strings.NewReader("Mamamia!\nAnother Line"), &stdout, io.Discard); err != nil {
			panic(err)
		}

		log.Println(stdout.String())
	}

	{
		filename := "/tmp/temp_file.txt"

		defer func() {
			if err := d.ExecuteCommand(ctx, []string{"rm", "-f", filename}, nil, io.Discard, io.Discard); err != nil {
				panic(err)
			}
		}()

		if err := driver.WriteFileContent(ctx, d, filename, strings.NewReader("Desired\nFile\nContent")); err != nil {
			panic(err)
		}

		stdout := bytes.Buffer{}

		if err := driver.ReadFileContent(ctx, d, filename, &stdout); err != nil {
			panic(err)
		}

		log.Println(stdout.String())
	}
}
