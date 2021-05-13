// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package emailutil

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// NOTE: regexp also used in src/views/detective.vue
	emailRegexp     = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)
	ErrInvalidEmail = errors.New("Not a valid email address")
)

func IsValidEmailAddress(email string) bool {
	return emailRegexp.Match([]byte(email))
}

func Split(email string) (local string, domain string, err error) {
	if !IsValidEmailAddress(email) {
		return "", "", ErrInvalidEmail
	}

	emailParts := strings.Split(email, "@")

	if len(emailParts) != 2 {
		return "", "", errors.New("Can't split email address")
	}

	return emailParts[0], emailParts[1], nil
}
