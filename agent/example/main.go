// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"bytes"
	"context"
	"gitlab.com/lightmeter/controlcenter/agent/driver"
	"gitlab.com/lightmeter/controlcenter/agent/parser"
	"gitlab.com/lightmeter/controlcenter/agent/postfix"
	"log"
	"os"
	"strings"
)

func buildDriver() (driver.Driver, error) {
	container, useDocker := os.LookupEnv("POSTFIX_CONTAINER")

	if !useDocker {
		return &driver.LocalDriver{}, nil
	}

	return driver.NewDockerDriver(container, "0")
}

func main() {
	d, err := buildDriver()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// For debug only
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	driver.Stdout = &stdout
	driver.Stderr = &stderr

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Stdout: > %s", stdout.String())
			log.Printf("Stderr: > %s", stderr.String())
			panic(r)
		}
	}()

	//basicDriverStuff(ctx, d)

	configurePostfix(ctx, d)
}

func configurePostfix(ctx context.Context, d driver.Driver) {
	if err := postfix.BlockIPs(ctx, d, []string{"123.456.789", "221.221.221.221"}); err != nil {
		panic(err)
	}
}

func basicDriverStuff(ctx context.Context, d driver.Driver) {
	{
		stdout := bytes.Buffer{}

		if err := d.ExecuteCommand(ctx, []string{"postconf"}, nil, &stdout, nil); err != nil {
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
		if err := d.ExecuteCommand(ctx, []string{"rev"}, strings.NewReader("Mamamia!\nAnother Line"), &stdout, nil); err != nil {
			panic(err)
		}

		log.Println(stdout.String())
	}

	{
		filename := "/tmp/temp_file.txt"

		defer func() {
			if err := d.ExecuteCommand(ctx, []string{"rm", "-f", filename}, nil, nil, nil); err != nil {
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
