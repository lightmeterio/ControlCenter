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
	Convey("Valid email addresses", t, func() {
		// valid emails
		So(IsValidEmailAddress("hello@lightmeter.io"), ShouldBeTrue)
		So(IsValidEmailAddress("hello+test@lightmeter.io"), ShouldBeTrue)

		// invalid emails
		So(IsValidEmailAddress("lightmeter.io"), ShouldBeFalse)
		So(IsValidEmailAddress("@lightmeter.io"), ShouldBeFalse)
		So(IsValidEmailAddress("hello+test@"), ShouldBeFalse)
	})

	Convey("Split complete email addresses", t, func() {
		user, domain, err := Split("hello@lightmeter.io")
		So(err, ShouldBeNil)
		So(user, ShouldEqual, "hello")
		So(domain, ShouldEqual, "lightmeter.io")

		user, domain, err = Split("hello+test@lightmeter.io")
		So(err, ShouldBeNil)
		So(user, ShouldEqual, "hello+test")
		So(domain, ShouldEqual, "lightmeter.io")

		_, _, err = Split("not-an-email")
		So(err, ShouldEqual, ErrPartialEmail)

		_, _, err = Split("test@example@org")
		So(err, ShouldEqual, ErrInvalidEmail)

		_, _, err = Split("lightmeter.io")
		So(err, ShouldEqual, ErrPartialEmail)

		_, _, err = Split("@lightmeter.io")
		So(err, ShouldEqual, ErrPartialEmail)
	})

	Convey("Split partial email addresses", t, func() {
		user, domain, partial, err := SplitPartial("hello@lightmeter.io")
		So(err, ShouldBeNil)
		So(partial, ShouldBeFalse)
		So(user, ShouldEqual, "hello")
		So(domain, ShouldEqual, "lightmeter.io")

		user, domain, partial, err = SplitPartial("hello+test@lightmeter.io")
		So(err, ShouldBeNil)
		So(partial, ShouldBeFalse)
		So(user, ShouldEqual, "hello+test")
		So(domain, ShouldEqual, "lightmeter.io")

		user, domain, partial, err = SplitPartial("@lightmeter.io")
		So(err, ShouldBeNil)
		So(partial, ShouldBeTrue)
		So(user, ShouldEqual, "")
		So(domain, ShouldEqual, "lightmeter.io")

		user, domain, partial, err = SplitPartial("lightmeter.io")
		So(err, ShouldBeNil)
		So(partial, ShouldBeTrue)
		So(user, ShouldEqual, "")
		So(domain, ShouldEqual, "lightmeter.io")

		user, domain, partial, err = SplitPartial("test")
		So(err, ShouldBeNil)
		So(partial, ShouldBeTrue)
		So(user, ShouldEqual, "")
		So(domain, ShouldEqual, "test")

		_, _, _, err = SplitPartial("test@example@org")
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
