// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package lmsqlite3

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
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

func TestParsingTimeFromJSON(t *testing.T) {
	Convey("Time from JSON", t, func() {
		Convey("Invalid format", func() {
			_, err := jsonTimeToTimestamp(`invalid`)
			So(err, ShouldNotBeNil)
		})

		Convey("Postgres format on encoding with json_* functions", func() {
			t, err := jsonTimeToTimestamp(`2021-10-25T23:59:59`)
			So(err, ShouldBeNil)
			So(t, ShouldResemble, timeutil.MustParseTime(`2021-10-25 23:59:59 +0000`).Unix())
		})

		Convey("Go format, seconds only", func() {
			t, err := jsonTimeToTimestamp(`2021-10-25T23:59:59Z`)
			So(err, ShouldBeNil)
			So(t, ShouldResemble, timeutil.MustParseTime(`2021-10-25 23:59:59 +0000`).Unix())
		})

		Convey("Go format, with nanoseconds", func() {
			t, err := jsonTimeToTimestamp(`2021-10-25T23:59:59.000000Z`)
			So(err, ShouldBeNil)
			So(t, ShouldResemble, timeutil.MustParseTime(`2021-10-25 23:59:59 +0000`).Unix())
		})
	})
}

func TestHostDomain(t *testing.T) {
	Convey("Obtain the host domain from a domain", t, func() {
		Convey("Unmanaged TLD, no subdomain", func() {
			d, err := hostDomainFromDomain(`localhost`)
			So(err, ShouldBeNil)
			So(d, ShouldEqual, `localhost`)
		})

		Convey("Unmanaged TLD", func() {
			d, err := hostDomainFromDomain(`there.is.no.such-tld`)
			So(err, ShouldBeNil)
			So(d, ShouldEqual, `no.such-tld`)
		})

		Convey("Unmanaged TLD, Fix case", func() {
			d, err := hostDomainFromDomain(`THERE.Is.No.SUCh-TLD`)
			So(err, ShouldBeNil)
			So(d, ShouldEqual, `no.such-tld`)
		})

		Convey("google.com", func() {
			d, err := hostDomainFromDomain(`ALT2.ASPMX.L.GOOGLE.com`)
			So(err, ShouldBeNil)
			So(d, ShouldEqual, `google.com`)
		})

		Convey("google.com.br", func() {
			d, err := hostDomainFromDomain(`ALT2.ASPMX.L.GOOGLE.com.br`)
			So(err, ShouldBeNil)
			So(d, ShouldEqual, `google.com.br`)
		})

		Convey("IP address returns itself", func() {
			d, err := hostDomainFromDomain(`11.22.33.44`)
			So(err, ShouldBeNil)
			So(d, ShouldEqual, `11.22.33.44`)
		})
	})
}
