// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package version

import "fmt"

var (
	Version     string
	TagOrBranch string
	Commit      string
)

func PrintVersion() {
	//nolint:forbidigo
	fmt.Printf("Lightmeter ControlCenter %s\n", Version)
}
