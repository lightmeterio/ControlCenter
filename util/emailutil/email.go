// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package emailutil

import (
	"errors"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"golang.org/x/net/publicsuffix"
	"net"
	"regexp"
	"strings"
)

var (
	// NOTE: valid-email-regexp also used in src/views/detective.vue
	localRe         = `[^@\s]+`
	domainRe        = `[^@\s]+`
	domainRegexp    = regexp.MustCompile(fmt.Sprintf(`^%s$`, domainRe))
	emailRegexp     = regexp.MustCompile(fmt.Sprintf(`^%s@%s$`, localRe, domainRe))
	ErrInvalidEmail = errors.New("Not a valid email address")
	ErrPartialEmail = errors.New("Email address isn't complete")
)

func IsValidEmailAddress(email string) bool {
	return emailRegexp.Match([]byte(email))
}

func Split(email string) (local string, domain string, err error) {
	local, domain, isPartial, err := SplitPartial(email)

	if err != nil {
		return "", "", err
	}

	if isPartial {
		return "", "", ErrPartialEmail
	}

	return local, domain, nil
}

func SplitPartial(email string) (local string, domain string, isPartial bool, err error) {
	if domainRegexp.Match([]byte(email)) {
		return "", email, true, nil
	}

	emailParts := strings.Split(email, "@")
	if len(emailParts) != 2 {
		return "", "", false, ErrInvalidEmail
	}

	isPartial = len(emailParts[0]) == 0

	if !domainRegexp.Match([]byte(emailParts[1])) {
		return "", "", false, ErrInvalidEmail
	}

	return emailParts[0], emailParts[1], isPartial, nil
}

func HasMX(email string) bool {
	_, domain, err := Split(email)

	if err != nil {
		return false
	}

	mxs, err := net.LookupMX(domain)

	return err == nil && len(mxs) > 0
}

func IsDisposableEmailAddress(email string) bool {
	_, domain, err := Split(email)

	if err != nil {
		return false
	}

	_, isDisposable := disposableDomains[domain]

	return isDisposable
}

func HostDomainFromDomain(domain string) (string, error) {
	// not a domain, but an IP address!
	if ip := net.ParseIP(domain); ip != nil {
		return domain, nil
	}

	d, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if len(d) == 0 {
		return strings.ToLower(domain), nil
	}

	if err != nil {
		return "", errorutil.Wrap(err)
	}

	return strings.ToLower(d), nil
}
