// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	"gitlab.com/lightmeter/controlcenter/deliverydb"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type logsRunner struct {
	runner.CancellableRunner

	tracker    *tracking.Tracker
	deliveries *deliverydb.DB
}

func newLogsRunner(tracker *tracking.Tracker, deliveries *deliverydb.DB) *logsRunner {
	r := &logsRunner{deliveries: deliveries, tracker: tracker}

	r.CancellableRunner = runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		trackerDone, trackerCancel := runner.Run(tracker)
		deliveriesDone, deliveriesCancel := runner.Run(deliveries)

		go func() {
			<-cancel
			trackerCancel()
			errorutil.MustSucceed(trackerDone())
			deliveriesCancel()
		}()

		go func() {
			errorutil.MustSucceed(deliveriesDone())
			done <- nil
		}()
	})

	return r
}
