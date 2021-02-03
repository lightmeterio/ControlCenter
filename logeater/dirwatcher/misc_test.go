// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package dirwatcher

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
)

func TestSortableRecords(t *testing.T) {
	Convey("Test Order", t, func() {
		So(
			sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{}}.Less(
				sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:01 +0000`), record: parsedRecord{}}),
			ShouldBeTrue)

		So(
			sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:01 +0000`), record: parsedRecord{}}.Less(
				sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{}}),
			ShouldBeFalse)

		So(
			sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{}}.Less(
				sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{}}),
			ShouldBeFalse)

		So(
			sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{queueIndex: 1}}.Less(
				sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{queueIndex: 2}}),
			ShouldBeTrue)

		So(
			sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{queueIndex: 2}}.Less(
				sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{queueIndex: 1}}),
			ShouldBeFalse)

		So(
			sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{queueIndex: 1}}.Less(
				sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{queueIndex: 1}}),
			ShouldBeFalse)

		So(
			sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{queueIndex: 1, sequence: 1}}.Less(
				sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{queueIndex: 1, sequence: 2}}),
			ShouldBeTrue)

		So(
			sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{queueIndex: 1, sequence: 2}}.Less(
				sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{queueIndex: 1, sequence: 1}}),
			ShouldBeFalse)

		So(
			sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{queueIndex: 1, sequence: 1}}.Less(
				sortableRecord{time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`), record: parsedRecord{queueIndex: 1, sequence: 1}}),
			ShouldBeFalse)
	})
}
