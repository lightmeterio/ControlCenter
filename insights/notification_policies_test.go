// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/detectiveescalation"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"testing"
)

func TestNotificationPolicies(t *testing.T) {
	Convey("Test Notification Policies", t, func() {
		policies := DefaultNotificationPolicy{}

		Convey("Bad Insights are not rejected", func() {
			r, err := policies.Reject(notificationCore.Notification{Content: core.InsightProperties{Rating: core.BadRating}})
			So(err, ShouldBeNil)
			So(r, ShouldBeFalse)
		})

		Convey("Other ratings are rejected", func() {
			for _, r := range []core.Rating{core.GoodRating, core.Unrated, core.OkRating} {
				r, err := policies.Reject(notificationCore.Notification{Content: core.InsightProperties{Rating: r}})
				So(err, ShouldBeNil)
				So(r, ShouldBeTrue)
			}
		})

		Convey("Detective escalated insights are not rejected", func() {
			r, err := policies.Reject(notificationCore.Notification{
				Content: detectiveescalation.BuildInsightProperties(&timeutil.FakeClock{}, detectiveescalation.SampleInsightContent),
			})

			So(err, ShouldBeNil)
			So(r, ShouldBeFalse)
		})

	})
}
