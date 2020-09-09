package api

import (
	"encoding/json"
	"fmt"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	mock_insights_fetcher "gitlab.com/lightmeter/controlcenter/insights/core/mock"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeFetchedInsight struct {
	id          int
	time        time.Time
	priority    core.Priority
	category    core.Category
	contentType string
	content     core.Content
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

func (f *fakeFetchedInsight) Priority() core.Priority {
	return f.priority
}

func (f *fakeFetchedInsight) ContentType() string {
	return f.contentType
}

func (f *fakeFetchedInsight) Content() core.Content {
	return f.content
}

func TestInsights(t *testing.T) {
	ctrl := gomock.NewController(t)

	f := mock_insights_fetcher.NewMockFetcher(ctrl)

	parseTimeInterval := func(from, to string) data.TimeInterval {
		i, err := data.ParseTimeInterval(from, to, time.UTC)

		if err != nil {
			panic("invalid time interval!!!")
		}

		return i
	}

	Convey("Test Insights", t, func() {
		Convey("Missing mandatory arguments", func() {
			s := httptest.NewServer(fetchInsightsHandler{f: f, timezone: time.UTC})
			r, err := http.Get(s.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Number of entries cannot be negative", func() {
			s := httptest.NewServer(fetchInsightsHandler{f: f, timezone: time.UTC})
			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31&order=creationDesc&entries=-42", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Get some insights", func() {
			f.EXPECT().FetchInsights(core.FetchOptions{
				Interval:   parseTimeInterval(`1999-01-01`, `1999-12-31`),
				OrderBy:    core.OrderByCreationDesc,
				FilterBy:   core.NoFetchFilter,
				Category:   core.NoCategory,
				MaxEntries: 0,
			}).Return([]core.FetchedInsight{
				&fakeFetchedInsight{
					id:          1,
					category:    core.InfoCategory,
					content:     "content1",
					contentType: "fake_content_1",
					priority:    2,
					time:        time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				&fakeFetchedInsight{
					id:          2,
					category:    core.WarningCategory,
					content:     "content2",
					contentType: "fake_content_2",
					priority:    4,
					time:        time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC),
				},
			}, nil)

			s := httptest.NewServer(fetchInsightsHandler{f: f, timezone: time.UTC})
			r, err := http.Get(fmt.Sprintf("%s?from=1999-01-01&to=1999-12-31&order=creationDesc", s.URL))
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)

			var body []interface{}
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&body)
			So(err, ShouldBeNil)

			So(body, ShouldResemble, []interface{}{
				map[string]interface{}{
					"Category":    "info",
					"Content":     "content1",
					"ContentType": "fake_content_1",
					"ID":          float64(1),
					"Priority":    float64(2),
					"Time":        "1999-01-01T00:00:00Z",
				},
				map[string]interface{}{
					"Category":    "warning",
					"Content":     "content2",
					"ContentType": "fake_content_2",
					"ID":          float64(2),
					"Priority":    float64(4),
					"Time":        "1999-12-31T00:00:00Z",
				},
			})
		})
	})

	ctrl.Finish()
}
