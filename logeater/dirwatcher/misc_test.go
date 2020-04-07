package dirwatcher

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSortableRecords(t *testing.T) {
	Convey("Test Order", t, func() {
		So(
			sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}}.Less(
				sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:01 +0000`)}}),
			ShouldBeTrue)

		So(
			sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:01 +0000`)}}.Less(
				sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}}),
			ShouldBeFalse)

		So(
			sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}}.Less(
				sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}}),
			ShouldBeFalse)

		So(
			sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}, queueIndex: 1}.Less(
				sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}, queueIndex: 2}),
			ShouldBeTrue)

		So(
			sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}, queueIndex: 2}.Less(
				sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}, queueIndex: 1}),
			ShouldBeFalse)

		So(
			sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}, queueIndex: 1}.Less(
				sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}, queueIndex: 1}),
			ShouldBeFalse)

		So(
			sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}, queueIndex: 1, sequence: 1}.Less(
				sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}, queueIndex: 1, sequence: 2}),
			ShouldBeTrue)

		So(
			sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}, queueIndex: 1, sequence: 2}.Less(
				sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}, queueIndex: 1, sequence: 1}),
			ShouldBeFalse)

		So(
			sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}, queueIndex: 1, sequence: 1}.Less(
				sortableRecord{record: timedRecord{time: parseTime(`2000-01-01 00:00:00 +0000`)}, queueIndex: 1, sequence: 1}),
			ShouldBeFalse)
	})
}
