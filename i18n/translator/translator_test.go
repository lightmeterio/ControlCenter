// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package translator

import (
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/text/language"
	"golang.org/x/text/message/catalog"
	"testing"
	"time"
)

func TestTranslators(t *testing.T) {
	Convey("Test Translators", t, func() {
		b := catalog.NewBuilder(catalog.Fallback(language.English))
		translators := New(b)

		Convey("No strings translated. Get value itself", func() {
			t := translators.Translator(language.English, time.Time{})
			s, err := t.Translate("Hello")
			So(err, ShouldBeNil)
			So(s, ShouldEqual, "Hello")
		})

		b.SetString(language.Spanish, "Hello World!", "Hola Mundo!")
		b.SetString(language.English, "Hello World!", "Hello World!")

		Convey("Translate to the same value", func() {
			t := translators.Translator(language.English, time.Time{})
			s, err := t.Translate("Hello World!")
			So(err, ShouldBeNil)
			So(s, ShouldEqual, "Hello World!")
		})

		Convey("Translate to another language", func() {
			t := translators.Translator(language.Spanish, time.Time{})
			s, err := t.Translate("Hello World!")
			So(err, ShouldBeNil)
			So(s, ShouldEqual, "Hola Mundo!")
		})
	})
}
