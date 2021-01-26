package workspace

import (
	"gitlab.com/lightmeter/controlcenter/deliverydb"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type logsRunner struct {
	runner.CancelableRunner

	tracker    *tracking.Tracker
	deliveries *deliverydb.DB
}

func newLogsRunner(tracker *tracking.Tracker, deliveries *deliverydb.DB) *logsRunner {
	r := &logsRunner{deliveries: deliveries, tracker: tracker}

	r.CancelableRunner = runner.NewCancelableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		trackerDone, trackerCancel := tracker.Run()
		deliveriesDone, deliveriesCancel := deliveries.Run()

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
