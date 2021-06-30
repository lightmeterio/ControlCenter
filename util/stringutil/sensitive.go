// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package stringutil

type Sensitive *string

func MakeSensitive(s string) Sensitive {
	var sensitive Sensitive = &s
	return sensitive
}
