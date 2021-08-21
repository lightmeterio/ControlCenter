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
)

func main() {
	driver, err := driver.NewDockerDriver(os.Getenv("POSTFIX_CONTAINER"), "0")
	if err != nil {
		panic(err)
	}

	var (
		stdout bytes.Buffer
	)

	if err := driver.ExecuteCommand(context.Background(), []string{"postconf"}, &stdout, io.Discard); err != nil {
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
