// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"net"
)

// parse IP address, but don't trigger error on special value "unknown"
func parseIP(b string) (net.IP, error) {
	if len(b) == 0 {
		return nil, nil
	}

	if b == "unknown" {
		return nil, nil
	}

	ip := net.ParseIP(b)

	if ip == nil {
		return nil, &net.ParseError{Type: "IP Address", Text: "Invalid IP"}
	}

	return ip, nil
}
