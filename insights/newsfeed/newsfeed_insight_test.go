package newsfeed

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	_ "gitlab.com/lightmeter/controlcenter/insights/migrations"
	insighttestsutil "gitlab.com/lightmeter/controlcenter/insights/testutil"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/template"
	"time"
)

var (
	dummyContext = context.Background()
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

type fakeRssContent struct {
	Title       string
	Description string
	Link        string
	Published   string
}

type fakeRssNewsResponse struct {
	time       time.Time
	shouldFail bool
	callTimes  int
	updated    string
	items      []fakeRssContent
}

type fakeRssHandler struct {
	clock     *insighttestsutil.FakeClock
	responses []*fakeRssNewsResponse
}

func (h *fakeRssHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	now := h.clock.Now()

	response, ok := func() (*fakeRssNewsResponse, bool) {
		for i := len(h.responses) - 1; i >= 0; i-- {
			v := h.responses[i]

			if !now.Before(v.time) {
				return v, true
			}
		}

		return nil, false
	}()

	if !ok {
		return
	}

	response.callTimes++

	if response.shouldFail {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	contentTemplate, err := template.New("feed").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:thr="http://purl.org/syndication/thread/1.0" xml:lang="en-US" xml:base="https://lightmeter.io/wp-atom.php">
  <title type="text">Some News Source</title>
  <subtitle type="text">Email deliverability for servers</subtitle>
	<updated>{{.Updated}}</updated>
  <id>https://lightmeter.io/feed/atom/</id>
  <link rel="self" type="application/atom+xml" href="https://lightmeter.io/category/releases/feed/atom/"/>
  <generator uri="https://wordpress.org/" version="5.6">WordPress</generator>
	{{range .Entries}}
  <entry>
    <author>
      <name>Author I Am</name>
    </author>
    <title type="html"><![CDATA[{{.Title}}]]></title>
    <link rel="alternate" type="text/html" href="{{.Link}}"/>
    <id>{{.Link}}</id>
    <updated>{{.Published}}</updated>
    <published>{{.Published}}</published>
    <category scheme="https://lightmeter.io" term="chosen_category"/>
    <summary type="html"><![CDATA[{{.Description}}]]></summary>
    <content type="html" xml:base="{{.Link}}"><![CDATA[Some Useless Content here, not used by the insight]]></content>
	</entry>
	{{end}}
</feed>`)

	errorutil.MustSucceed(err)

	values := struct {
		Updated string
		Entries []fakeRssContent
	}{
		Updated: response.updated,
		Entries: response.items,
	}

	err = contentTemplate.Execute(w, values)
	errorutil.MustSucceed(err)
}

func TestNewsFeedInsights(t *testing.T) {
	Convey("Newsfeed insight", t, func() {
		clock := &insighttestsutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)}

		h := &fakeRssHandler{clock: clock, responses: []*fakeRssNewsResponse{
			&fakeRssNewsResponse{
				time:       testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				shouldFail: false,
				updated:    `1999-12-20T00:00:00Z`,
				items: []fakeRssContent{
					{
						Title:       `Some First News`,
						Description: `Description of the First Item`,
						Link:        `https://example.com/news/1`,
						Published:   `1999-12-20T00:00:00Z`,
					},
				},
			},
			&fakeRssNewsResponse{
				// a request fails
				time:       testutil.MustParseTime(`2000-01-02 12:00:00 +0000`),
				shouldFail: true,
			},

			&fakeRssNewsResponse{
				// and 30 seconds later it succeeds
				time:       testutil.MustParseTime(`2000-01-02 12:00:30 +0000`),
				shouldFail: false,
				updated:    `2000-01-02T11:59:00Z`,
				items: []fakeRssContent{
					{
						Title:       `The Second News`,
						Description: `Another description`,
						Link:        `https://example.com/news/2`,
						Published:   `2000-01-02T11:59:00Z`,
					},
					{
						Title:       `Some First News`,
						Description: `Description of the First Item`,
						Link:        `https://example.com/news/1`,
						Published:   `1999-12-20T00:00:00Z`,
					},
				},
			},
		}}

		s := httptest.NewServer(h)

		accessor, clear := insighttestsutil.NewFakeAccessor(t)
		defer clear()

		Convey("Insight is generated only once", func() {
			detector := NewDetector(accessor, core.Options{"newsfeed": Options{
				URL:            s.URL,
				UpdateInterval: time.Hour * 2,
				RetryTime:      time.Second * 30,
			}})

			insighttestsutil.ExecuteCyclesUntil(detector, accessor, clock, testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour*48), time.Second*2)

			So(accessor.Insights, ShouldResemble, []int64{1, 2})

			insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-01 00:00:00 +0000`).Add(time.Hour * 48),
			}, OrderBy: core.OrderByCreationAsc})

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 2)

			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, "newsfeed_content")
			So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-01 00:00:00 +0000`))
			So(insights[0].Content(), ShouldResemble, &Content{
				Title:       `Some First News`,
				Description: `Description of the First Item`,
				Link:        `https://example.com/news/1`,
				Published:   testutil.MustParseTime(`1999-12-20 00:00:00 +0000`),
				GUID:        `https://example.com/news/1`,
			})

			So(insights[1].ID(), ShouldEqual, 2)
			So(insights[1].ContentType(), ShouldEqual, "newsfeed_content")
			So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-02 12:00:30 +0000`))
			So(insights[1].Content(), ShouldResemble, &Content{
				Title:       `The Second News`,
				Description: `Another description`,
				Link:        `https://example.com/news/2`,
				Published:   testutil.MustParseTime(`2000-01-02 11:59:00 +0000`),
				GUID:        `https://example.com/news/2`,
			})
		})
	})
}
