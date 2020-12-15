// SPDX-FileCopyrightText: 2020,  Lightmeter <hello@lightmeter.io>
// SPDX-License-Identifier: AGPL-3.0

package emailutil

import (
	"regexp"
)

var (
	emailRegexp = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)
)

func IsValidEmailAddress(email string) bool {
	return emailRegexp.Match([]byte(email))
}
