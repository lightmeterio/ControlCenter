// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"gitlab.com/lightmeter/controlcenter/config"
)

func main() {
	_, err := config.Parse([]string{"-help"}, func(string) (string, bool) { return "", false })
	if err != nil {
		panic(err)
	}
}
