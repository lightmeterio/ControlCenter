// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

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
<rss version="2.0"
        xmlns:content="http://purl.org/rss/1.0/modules/content/"
        xmlns:wfw="http://wellformedweb.org/CommentAPI/"
        xmlns:dc="http://purl.org/dc/elements/1.1/"
        xmlns:atom="http://www.w3.org/2005/Atom"
        xmlns:sy="http://purl.org/rss/1.0/modules/syndication/"
        xmlns:slash="http://purl.org/rss/1.0/modules/slash/"
        xmlns:lightmeter="http://lightmeter.io/rss/controlcenter">
  <channel>
    <title>Control Center News Insights – Lightmeter</title>
    <atom:link href="https://lightmeter.io/category/news-insights/feed/" rel="self" type="application/rss+xml"/>
    <link>https://lightmeter.io</link>
    <description>Email deliverability for servers</description>
    <lastBuildDate>{{.Updated}}</lastBuildDate>
    <language>en-US</language>
    <sy:updatePeriod>hourly</sy:updatePeriod>
    <sy:updateFrequency>1</sy:updateFrequency>
    <generator>https://wordpress.org/?v=5.6</generator>
    {{range .Entries}}
    <item>
      <title>{{.Title}}</title>
      <link>{{.Link}}</link>
      <dc:creator><![CDATA[Author Here]]></dc:creator>
      <pubDate>{{.Published}}</pubDate>
      <category><![CDATA[Control Center News Insights]]></category>
      <guid isPermaLink="false">{{.Link}}</guid>
      <description><![CDATA[unused description]]></description>
      <content:encoded><![CDATA[unused content]]></content:encoded>
      <lightmeter:newsInsightDescription>{{.Description}}</lightmeter:newsInsightDescription>
    </item>
    {{end}}
  </channel>
</rss>`)

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

func buildFakeHandlerWithClock(clock *insighttestsutil.FakeClock) *fakeRssHandler {
	return &fakeRssHandler{clock: clock, responses: []*fakeRssNewsResponse{
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

		&fakeRssNewsResponse{
			time:       testutil.MustParseTime(`2000-01-05 12:00:00 +0000`),
			shouldFail: false,
			updated:    `2000-01-05T12:00:00Z`,
			items: []fakeRssContent{
				{
					Title:       `The Third News`,
					Description: `Some Third Description`,
					Link:        `https://example.com/news/3`,
					Published:   `2000-01-05T11:59:00Z`,
				},
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

		&fakeRssNewsResponse{
			time:       testutil.MustParseTime(`2000-01-06 12:00:00 +0000`),
			shouldFail: false,
			updated:    `2000-01-06T12:00:00Z`,
			items: []fakeRssContent{
				{
					Title:       `The Forth News`,
					Description: `Some Forth Description`,
					Link:        `https://example.com/news/4`,
					Published:   `2000-01-06T11:59:00Z`,
				},
				{
					Title:       `The Third News`,
					Description: `Some Third Description`,
					Link:        `https://example.com/news/3`,
					Published:   `2000-01-05T11:59:00Z`,
				},
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
}

func TestNewsFeedInsights(t *testing.T) {
	Convey("Newsfeed insight", t, func() {
		accessor, clear := insighttestsutil.NewFakeAccessor(t)
		defer clear()

		Convey("Each insight is generated only once", func() {
			clock := &insighttestsutil.FakeClock{Time: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)}
			s := httptest.NewServer(buildFakeHandlerWithClock(clock))

			detector := NewDetector(accessor, core.Options{"newsfeed": Options{
				URL:            s.URL,
				UpdateInterval: time.Hour * 2,
				RetryTime:      time.Second * 30,
				// A very high time, with the intention of fetching all
				TimeLimit: time.Hour * 24 * 300,
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

		Convey("Use feed entries only from the past two days", func() {
			clock := &insighttestsutil.FakeClock{Time: testutil.MustParseTime(`2000-01-05 00:00:00 +0000`)}
			s := httptest.NewServer(buildFakeHandlerWithClock(clock))

			detector := NewDetector(accessor, core.Options{"newsfeed": Options{
				URL:            s.URL,
				UpdateInterval: time.Hour * 2,
				RetryTime:      time.Second * 30,
				TimeLimit:      time.Hour * 24 * 2,
			}})

			insighttestsutil.ExecuteCyclesUntil(detector, accessor, clock, testutil.MustParseTime(`2000-01-06 23:00:00 +0000`), time.Second*2)

			So(accessor.Insights, ShouldResemble, []int64{1, 2})

			insights, err := accessor.FetchInsights(dummyContext, core.FetchOptions{Interval: data.TimeInterval{
				From: testutil.MustParseTime(`2000-01-01 00:00:00 +0000`),
				To:   testutil.MustParseTime(`2000-01-10 00:00:00 +0000`),
			}, OrderBy: core.OrderByCreationAsc})

			So(err, ShouldBeNil)

			So(len(insights), ShouldEqual, 2)

			So(insights[0].ID(), ShouldEqual, 1)
			So(insights[0].ContentType(), ShouldEqual, "newsfeed_content")
			So(insights[0].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-05 12:00:00 +0000`))
			So(insights[0].Content(), ShouldResemble, &Content{
				Title:       `The Third News`,
				Description: `Some Third Description`,
				Link:        `https://example.com/news/3`,
				Published:   testutil.MustParseTime(`2000-01-05 11:59:00 +0000`),
				GUID:        `https://example.com/news/3`,
			})

			So(insights[1].ID(), ShouldEqual, 2)
			So(insights[1].ContentType(), ShouldEqual, "newsfeed_content")
			So(insights[1].Time(), ShouldEqual, testutil.MustParseTime(`2000-01-06 12:00:00 +0000`))
			So(insights[1].Content(), ShouldResemble, &Content{
				Title:       `The Forth News`,
				Description: `Some Forth Description`,
				Link:        `https://example.com/news/4`,
				Published:   testutil.MustParseTime(`2000-01-06 11:59:00 +0000`),
				GUID:        `https://example.com/news/4`,
			})
		})
	})
}
