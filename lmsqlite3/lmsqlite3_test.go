// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package lmsqlite3

import (
	. "github.com/smartystreets/goconvey/convey"
	"net"
	"testing"
)

// in the datase, ip addresses are stored as binary blobs
func strAsBytes(v string) []byte {
	ip := net.ParseIP(v)

	if len(ip) == 0 {
		panic("Invalid IP address")
	}

	return ip
}

func TestIpToStringSqliteFunction(t *testing.T) {
	Convey("IpToString", t, func() {
		So(ipToString(nil), ShouldEqual, "")
		So(ipToString([]byte{1, 2, 3}), ShouldEqual, "")
		So(ipToString(strAsBytes("127.0.0.1")), ShouldEqual, "127.0.0.1")
	})
}
