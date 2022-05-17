// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/config"
	"gitlab.com/lightmeter/controlcenter/detective/escalator"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/insights/detectiveescalation"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"gitlab.com/lightmeter/controlcenter/workspace"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func fetchInsights(ws *workspace.Workspace, cat core.Category) []core.FetchedInsight {
	fetcher := ws.InsightsFetcher()

	insights, err := fetcher.FetchInsights(context.Background(), core.FetchOptions{
		Interval: timeutil.MustParseTimeInterval("0000-01-01", "5000-01-01"),
		FilterBy: core.FilterByCategory,
		Category: cat,
	}, timeutil.RealClock{})

	So(err, ShouldBeNil)

	return insights
}

func TestMain(t *testing.T) {
	Convey("Test Main", t, func() {
		wsDir, clearWsDir := testutil.TempDir(t)
		defer clearWsDir()

		logsDir, clearLogsDir := testutil.TempDir(t)
		defer clearLogsDir()

		logsArchive, err := os.Open(path.Join("test_files", "postfix_logs", "complete.tar.gz"))
		So(err, ShouldBeNil)
		testutil.ExtractTarGz(logsArchive, logsDir)

		// Only import the logs in a workspace
		config := config.Config{
			WorkspaceDirectory: wsDir,
			DirsToWatch:        []string{path.Join(logsDir, "logs_sample")},
			ImportOnly:         true,
			LogFormat:          "default",
			MultiNodeType:      "single",
		}

		// first execution
		func() {
			ws, reader, err := buildWorkspaceAndLogReader(config)
			So(err, ShouldBeNil)
			defer ws.Close()
			done, cancel := runner.Run(ws)
			reader.Run()
			cancel()
			So(done(), ShouldBeNil)

			// ensure there are no detective insights
			for _, i := range fetchInsights(ws, core.LocalCategory) {
				So(i.ContentType(), ShouldNotEqual, detectiveescalation.ContentType)
			}
		}()

		// second execution, using the same workspace.
		// We create a new insight and it should appear here
		func() {
			ws, reader, err := buildWorkspaceAndLogReader(config)
			So(err, ShouldBeNil)
			defer ws.Close()
			done, cancel := runner.Run(ws)

			// Won't reimport the logs
			err = reader.Run()
			So(err, ShouldBeNil)

			// Request insight to be created
			ws.DetectiveEscalationRequester().Request(escalator.Request{
				Sender:    "h-a163e9c@h-ffeb17e38996c.com",
				Recipient: "h-291085e837e73@h-7db8006395b.com",
				Interval:  timeutil.MustParseTimeInterval("0000-01-01", "5000-01-01"),
			})

			// Sleep time enough for an insight cycle to be executed (2s)
			time.Sleep(4 * time.Second)

			cancel()
			So(done(), ShouldBeNil)

			numberOfDetectiveInsights := 0

			// ensure there are no detective insights
			for _, i := range fetchInsights(ws, core.LocalCategory) {
				if i.ContentType() == detectiveescalation.ContentType {
					numberOfDetectiveInsights++
				}
			}

			So(numberOfDetectiveInsights, ShouldEqual, 1)
		}()
	})
}
