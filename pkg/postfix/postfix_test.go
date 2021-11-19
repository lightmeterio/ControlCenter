// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postfix

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestChecksums(t *testing.T) {
	Convey("Test Sums", t, func() {
		hasher := NewHasher()

		Convey("Equal records have the same Sum", func() {
			So(
				ComputeChecksum(hasher, "Some content here"),
				ShouldEqual,
				ComputeChecksum(hasher, "Some content here"),
			)
		})

		Convey("Records with different content have different Sums", func() {
			So(
				ComputeChecksum(hasher, "Some content here"),
				ShouldNotEqual,
				ComputeChecksum(hasher, "Some different content here"),
			)
		})
	})
}
