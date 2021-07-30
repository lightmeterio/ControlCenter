// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package emailutil

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestDisposableDomains(t *testing.T) {
	Convey("Split email addresses", t, func() {
		user, domain, err := Split("hello@lightmeter.io")
		So(user, ShouldEqual, "hello")
		So(domain, ShouldEqual, "lightmeter.io")
		So(err, ShouldBeNil)

		user, domain, err = Split("not-an-email")
		So(err, ShouldEqual, ErrInvalidEmail)
	})

	Convey("MX records", t, func() {
		So(HasMX("hello@lightmeter.io"), ShouldBeTrue)
		So(HasMX("hi@dhsoiafhdifjsaiod.doh"), ShouldBeFalse)
	})

	Convey("Disposable email domains", t, func() {
		// disposable
		So(IsDisposableEmailAddress("anybody@0-180.com"), ShouldBeTrue)
		So(IsDisposableEmailAddress("anyone@mailinator.com"), ShouldBeTrue)
		So(IsDisposableEmailAddress("anytwo@yopmail.com"), ShouldBeTrue)
		So(IsDisposableEmailAddress("anywho@zzzz1717.com"), ShouldBeTrue)

		// non-disposable
		So(IsDisposableEmailAddress("user@gmail.com"), ShouldBeFalse)
		So(IsDisposableEmailAddress("john.doe@hotmail.com"), ShouldBeFalse)
		So(IsDisposableEmailAddress("hello@lightmeter.io"), ShouldBeFalse)
		So(IsDisposableEmailAddress("NutzerIn@gmx.de"), ShouldBeFalse)
		So(IsDisposableEmailAddress("utilisatrice@free.fr"), ShouldBeFalse)
		So(IsDisposableEmailAddress("utilisateur@orange.fr"), ShouldBeFalse)
	})
}
