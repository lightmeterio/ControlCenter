// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postconf

import (
	"bytes"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func parse(content []byte) (*Values, error) {
	buffer := bytes.NewBuffer(content)
	return Parse(buffer)
}

func TestPostconfParser(t *testing.T) {
	Convey("Postconf postconf", t, func() {
		complexExample :=
			`
smtpd_tls_eckey_file = $smtpd_tls_eccert_file
smtpd_tls_eccert_file = /some/path
syslog_name = ${multi_instance_name?{$multi_instance_name}:{postfix}}
smtpd_starttls_timeout = ${stress?{10}:{300}}s
mynetworks_style = ${{$compatibility_level} < {2} ? {subnet} : {host}}
`
		_ = complexExample

		Convey("Basic parsing", func() {
			conf, err := parse([]byte(`smtpd_tls_eckey_file = $smtpd_tls_eccert_file
smtpd_tls_eccert_file = /some/path
empty_value =
refers_to_empty_value = $empty_value
refers_to_inexistent_value = $inexistent_value
refers_to_link = $smtpd_tls_eckey_file`))
			So(err, ShouldBeNil)

			simpleValue, err := conf.Resolve("smtpd_tls_eccert_file")
			So(err, ShouldBeNil)
			So(simpleValue, ShouldEqual, "/some/path")

			variableValue, err := conf.Resolve("smtpd_tls_eckey_file")
			So(err, ShouldBeNil)
			So(variableValue, ShouldEqual, "/some/path")

			emptyValue, err := conf.Resolve("empty_value")
			So(err, ShouldBeNil)
			So(emptyValue, ShouldEqual, "")

			linkToEmptyValue, err := conf.Resolve("refers_to_empty_value")
			So(err, ShouldBeNil)
			So(linkToEmptyValue, ShouldEqual, "")

			_, err = conf.Resolve("non_existent_key")
			So(err, ShouldEqual, ErrKeyNotFound)

			linkToInexistentValue, err := conf.Resolve("refers_to_inexistent_value")
			So(err, ShouldBeNil)
			So(linkToInexistentValue, ShouldEqual, "")

			twoLevelLinkValue, err := conf.Resolve("refers_to_link")
			So(err, ShouldBeNil)
			So(twoLevelLinkValue, ShouldEqual, "/some/path")
		})

		Convey("Interpolated Values", func() {
			// TODO: implement Resolv() for such cases?
			conf, err := parse([]byte(`value1 = 3
value2 = mamamia
value3 = $value1 something ${value2}s
value4 = $value2`))
			So(err, ShouldBeNil)
			value3, err := conf.Value("value3")
			So(err, ShouldBeNil)
			So(value3, ShouldEqual, "$value1 something ${value2}s")

			value2, err := conf.Value("value2")
			So(err, ShouldBeNil)
			So(value2, ShouldEqual, "mamamia")

			value4, err := conf.Value("value4")
			So(err, ShouldEqual, nil)
			So(value4, ShouldEqual, "$value2")
		})
	})
}
