// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postfix

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
)

func TestChecksums(t *testing.T) {
	Convey("Test Sums", t, func() {
		hasher := NewHasher()

		Convey("Equal records have the same Sum", func() {
			So(
				ComputeChecksum(hasher, Record{
					Time: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
					Line: "Some content here",
				}),
				ShouldEqual,
				ComputeChecksum(hasher, Record{
					Time: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
					Line: "Some content here",
				}),
			)
		})

		Convey("Records with different content have different Sums", func() {
			So(
				ComputeChecksum(hasher, Record{
					Time: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
					Line: "Some content here",
				}),
				ShouldNotEqual,
				ComputeChecksum(hasher, Record{
					Time: timeutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
					Line: "Some different content here",
				}),
			)
		})
	})
}
