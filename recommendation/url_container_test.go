// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package recommendation

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestURLContainer(t *testing.T) {
	Convey("Test URL container", t, func() {

		urlContainer := NewURLContainer()

		Convey("empty value", func() {
			v := urlContainer.Get("k")
			So(v, ShouldBeEmpty)
		})

		Convey("non empty value Set", func() {
			urlContainer.Set("k", "v")
			v := urlContainer.Get("k")
			So(v, ShouldEqual, "v")
		})

		Convey("non empty value SetForEach", func() {
			urlContainer.SetForEach([]Link{{ID: "k", Link: "v"}})
			v := urlContainer.Get("k")
			So(v, ShouldEqual, "v")
		})
	})
}

