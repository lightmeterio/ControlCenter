// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"bytes"
	"context"
	"gitlab.com/lightmeter/controlcenter/agent/driver"
	"gitlab.com/lightmeter/controlcenter/agent/postfix"
	"log"
	"os"
)

func buildDriver() (driver.Driver, error) {
	container, useDocker := os.LookupEnv("POSTFIX_CONTAINER")

	if !useDocker {
		return &driver.LocalDriver{}, nil
	}

	return driver.NewDockerDriver(container, "0")
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("You must pass the IP addresses to be blocked")
	}

	ips := os.Args[1:]

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

	if err := postfix.BlockIPs(ctx, d, ips); err != nil {
		log.Printf("Stdout: > %s", stdout.String())
		log.Printf("Stderr: > %s", stderr.String())
		panic(err)
	}
}
