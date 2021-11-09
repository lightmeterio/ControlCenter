// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

func Fuzz(data []byte) int {
	_, _, err := Parse(string(data))

	if !IsRecoverableError(err) {
		return 0
	}

	return 1
}
