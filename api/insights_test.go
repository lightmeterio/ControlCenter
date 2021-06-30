// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"encoding/json"
	"fmt"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	mock_insights_fetcher "gitlab.com/lightmeter/controlcenter/insights/core/mock"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"gitlab.com/lightmeter/controlcenter/recommendation"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeFetchedInsight struct {
	id            int
	time          time.Time
	rating        core.Rating
	category      core.Category
	contentType   string
	content       core.Content
	userRating    *int
	userRatingOld bool
}

func (f *fakeFetchedInsight) ID() int {
	return f.id
}

func (f *fakeFetchedInsight) Time() time.Time {
	return f.time
}

func (f *fakeFetchedInsight) Category() core.Category {
	return f.category
}

func (f *fakeFetchedInsight) Rating() core.Rating {
	return f.rating
}

func (f *fakeFetchedInsight) ContentType() string {
	return f.contentType
}

func (f *fakeFetchedInsight) Content() core.Content {
	return f.content
}

func (f *fakeFetchedInsight) UserRating() *int {
	return f.userRating
}

func (f *fakeFetchedInsight) UserRatingOld() bool {
	return f.userRatingOld
}

type content struct {
	V           string `json:"v"`
	ContentType string `json:"content_type"`
}

func (content) Title() notificationCore.ContentComponent {
	return fakeComponent{}
}

func (content) Description() notificationCore.ContentComponent {
	return fakeComponent{}
}

func (content) Metadata() notificationCore.ContentMetadata {
	return nil
}

type fakeComponent struct{}

func (fakeComponent) String() string {
	return ""
}

func (fakeComponent) TplString() string {
	return ""
}

func (fakeComponent) Args() []interface{} {
	return nil
}

func (c content) HelpLink(container core.URLContainer) string {
	return container.Get(c.ContentType)
}

func parseTimeInterval(from, to string) timeutil.TimeInterval {
	i, err := timeutil.ParseTimeInterval(from, to, time.UTC)

	if err != nil {
		panic("invalid time interval!!!")
	}

	return i
}

func TestInsights(t *testing.T) {
	Convey("Test Insights", t, func() {
		urlContainer := recommendation.NewURLContainer()

		urlContainer.SetForEach([]recommendation.Link{
			{Link: "https://kb.lightemter.io/KB0001", ID: "message_rbl_Yahoo"},
			{Link: "https://kb.lightemter.io/KB0002", ID: "local_rbl_check"},
		})

		chain := httpmiddleware.New(httpmiddleware.RequestWithInterval(time.UTC))

		ctrl := gomock.NewController(t)

		defer ctrl.Finish()

		f := mock_insights_fetcher.NewMockFetcher(ctrl)

		Convey("Missing mandatory arguments", func() {
			s := httptest.NewServer(chain.WithEndpoint(fetchInsightsHandler{f: f, recommendationURLContainer: urlContainer}))
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusUnprocessableEntity)
		})

		Convey("Number of entries cannot be negative", func() {
			s := httptest.NewServer(chain.WithEndpoint(fetchInsightsHandler{f: f, recommendationURLContainer: urlContainer}))
			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31&order=creationDesc&entries=-42", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		contentType1 := "local_rbl_check"
		contentType2 := "message_rbl_Yahoo"

		Convey("Get some insights", func() {
			f.EXPECT().FetchInsights(gomock.Any(), core.FetchOptions{
				Interval:   parseTimeInterval(`1999-01-01`, `1999-12-31`),
				OrderBy:    core.OrderByCreationDesc,
				FilterBy:   core.NoFetchFilter,
				Category:   core.NoCategory,
				MaxEntries: 0,
			}, gomock.Any()).Return([]core.FetchedInsight{
				&fakeFetchedInsight{
					id:            1,
					category:      core.IntelCategory,
					content:       content{"content1", contentType1},
					contentType:   contentType1,
					rating:        core.BadRating,
					time:          time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
					userRatingOld: true,
				},
				&fakeFetchedInsight{
					id:            2,
					category:      core.LocalCategory,
					content:       content{"content2", contentType2},
					contentType:   contentType2,
					rating:        core.OkRating,
					time:          time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC),
					userRatingOld: true,
				},
			}, nil)

			s := httptest.NewServer(chain.WithEndpoint(fetchInsightsHandler{f: f, recommendationURLContainer: urlContainer}))
			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31&order=creationDesc", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body []interface{}
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&body)
			So(err, ShouldBeNil)

			So(body, ShouldResemble, []interface{}{
				map[string]interface{}{
					"category":        "intel",
					"content":         map[string]interface{}{"v": "content1", "content_type": contentType1},
					"content_type":    contentType1,
					"id":              float64(1),
					"rating":          "bad",
					"time":            "1999-01-01T00:00:00Z",
					"help_link":       "https://kb.lightemter.io/KB0002",
					"user_rating":     nil,
					"user_rating_old": true,
				},
				map[string]interface{}{
					"category":        "local",
					"content":         map[string]interface{}{"v": "content2", "content_type": contentType2},
					"content_type":    contentType2,
					"id":              float64(2),
					"rating":          "ok",
					"time":            "1999-12-31T00:00:00Z",
					"help_link":       "https://kb.lightemter.io/KB0001",
					"user_rating":     nil,
					"user_rating_old": true,
				},
			})
		})
	})
}

func TestImportProgress(t *testing.T) {
	Convey("Test ImportProgress", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		f := mock_insights_fetcher.NewMockProgressFetcher(ctrl)

		// process has not yet started. no progress
		f.EXPECT().Progress(gomock.Any()).Return(core.Progress{}, nil)

		// import starts, but no progress yet
		f.EXPECT().Progress(gomock.Any()).Return(core.Progress{Active: true}, nil)

		dynInt := func(v int) *int {
			return &v
		}

		dynTime := func(v time.Time) *time.Time {
			return &v
		}

		// import starts, some progress made
		f.EXPECT().Progress(gomock.Any()).Return(core.Progress{Active: true, Value: dynInt(15), Time: dynTime(testutil.MustParseTime(`2000-01-01 10:20:30 +0000`))}, nil)

		// import ended
		f.EXPECT().Progress(gomock.Any()).Return(core.Progress{Active: false, Value: dynInt(100), Time: dynTime(testutil.MustParseTime(`2000-01-01 10:30:00 +0000`))}, nil)

		chain := httpmiddleware.WithDefaultStackWithoutAuth()

		s := httptest.NewServer(chain.WithEndpoint(importProgressHandler{f: f}))

		getBody := func(r *http.Response) interface{} {
			var body interface{}
			dec := json.NewDecoder(r.Body)
			err := dec.Decode(&body)
			So(err, ShouldBeNil)
			return body
		}

		{
			// Import not yet started
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
			So(getBody(r), ShouldResemble, map[string]interface{}{"active": false})
		}

		{
			// starts, no progress yet
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
			So(getBody(r), ShouldResemble, map[string]interface{}{"active": true})
		}

		{
			// some progress
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
			So(getBody(r), ShouldResemble, map[string]interface{}{"active": true, "time": "2000-01-01T10:20:30Z", "value": float64(15)})
		}

		{
			// end progress
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
			So(getBody(r), ShouldResemble, map[string]interface{}{"active": false, "time": "2000-01-01T10:30:00Z", "value": float64(100)})
		}
	})
}
