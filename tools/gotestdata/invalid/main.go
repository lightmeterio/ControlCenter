// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

func I18n(s string) string {
	return ""
}

type DataC struct{}

func (d *DataC) String(s string) string {
	a := "DataC"
	b := I18n(a)

	return b
}

func main() {
	I18n("hello world")
}