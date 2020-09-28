package httpmiddleware

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"testing"
)

func TestGetTimeIntervalFromContext(t *testing.T) {

	Convey("Interval context", t, func() {
		Convey("invalid", func() {
			_, err := getIntervalFromContext(context.Background())
			So(err, ShouldNotBeNil)
		})

		Convey("valid", func() {
			ctx := context.Background()
			ctx = context.WithValue(ctx, Interval("interval"), data.TimeInterval{})
			_, err := getIntervalFromContext(ctx)
			So(err, ShouldBeNil)
		})
	})
}
