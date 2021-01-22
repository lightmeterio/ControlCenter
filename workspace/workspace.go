package workspace

import (
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/deliverydb"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/insights"
	insightsCore "gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/messagerbl"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/po"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"os"
	"path"
	"time"
)

type Workspace struct {
	runner.CancelableRunner

	deliveries     *deliverydb.DB
	tracker        *tracking.Tracker
	insightsEngine *insights.Engine
	auth           *auth.Auth
	rblDetector    *messagerbl.Detector
	rblChecker     localrbl.Checker

	NotificationCenter notification.Center

	settingsMetaHandler *meta.Handler
	settingsRunner      *meta.Runner

	closes closeutil.Closers
}

func NewWorkspace(workspaceDirectory string) (*Workspace, error) {
	if err := os.MkdirAll(workspaceDirectory, os.ModePerm); err != nil {
		return nil, errorutil.Wrap(err, "Error creating working directory ", workspaceDirectory)
	}

	deliveries, err := deliverydb.New(workspaceDirectory)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	tracker, err := tracking.New(workspaceDirectory, deliveries.ResultsPublisher())
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	auth, err := auth.NewAuth(workspaceDirectory, auth.Options{})

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	metadataConnPair, err := dbconn.NewConnPair(path.Join(workspaceDirectory, "master.db"))

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	m, err := meta.NewHandler(metadataConnPair, "master")

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	settingsRunner := meta.NewRunner(m)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	dashboard, err := dashboard.New(deliveries.ReadConnection())

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	translators := translator.New(po.DefaultCatalog)

	notificationCenter := notification.New(m.Reader, translators)

	rblChecker := localrbl.NewChecker(m.Reader, localrbl.Options{
		NumberOfWorkers:  10,
		Lookup:           localrbl.RealLookup,
		RBLProvidersURLs: localrbl.DefaultRBLs,
	})

	rblDetector := messagerbl.New(globalsettings.New(m.Reader))

	insightsEngine, err := insights.NewEngine(
		workspaceDirectory,
		notificationCenter, insightsOptions(dashboard, rblChecker, rblDetector))

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	logsRunner := newLogsRunner(tracker, deliveries)

	ws := &Workspace{
		deliveries:          deliveries,
		tracker:             tracker,
		insightsEngine:      insightsEngine,
		auth:                auth,
		rblDetector:         rblDetector,
		rblChecker:          rblChecker,
		settingsMetaHandler: m,
		settingsRunner:      settingsRunner,
		closes: closeutil.New(
			auth,
			tracker,
			deliveries,
			insightsEngine,
			m,
		),
		NotificationCenter: notificationCenter,
	}

	ws.CancelableRunner = runner.NewCancelableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		// Convert this one here to CancellableRunner!
		ws.rblChecker.StartListening()

		doneInsights, cancelInsights := ws.insightsEngine.Run()
		doneSettings, cancelSettings := ws.settingsRunner.Run()
		doneMsgRbl, cancelMsgRbl := ws.rblDetector.Run()

		doneLogsRunner, cancelLogsRunner := logsRunner.Run()

		go func() {
			<-cancel
			cancelLogsRunner()
			cancelMsgRbl()
			cancelSettings()
			cancelInsights()
		}()

		go func() {
			err := doneLogsRunner()
			errorutil.MustSucceed(err)

			doneMsgRbl()
			doneSettings()
			doneInsights()

			// TODO: return a combination of the "children" errors!
			done <- nil
		}()
	})

	return ws, nil
}

func (ws *Workspace) SettingsAcessors() (*meta.AsyncWriter, *meta.Reader) {
	return ws.settingsRunner.Writer(), ws.settingsMetaHandler.Reader
}

func (ws *Workspace) InsightsFetcher() insightsCore.Fetcher {
	return ws.insightsEngine.Fetcher()
}

func (ws *Workspace) Dashboard() (dashboard.Dashboard, error) {
	return dashboard.New(ws.deliveries.ReadConnection())
}

func (ws *Workspace) Auth() *auth.Auth {
	return ws.auth
}

func (ws *Workspace) MostRecentLogTime() time.Time {
	mostRecentDeliverTime := ws.deliveries.MostRecentLogTime()
	mostRecentTrackerTime := ws.tracker.MostRecentLogTime()

	if mostRecentTrackerTime.After(mostRecentDeliverTime) {
		return mostRecentTrackerTime
	}

	return mostRecentDeliverTime
}

func (ws *Workspace) NewPublisher() data.Publisher {
	return data.ComposedPublisher{ws.tracker.Publisher(), ws.rblDetector.NewPublisher()}
}

func (ws *Workspace) Close() error {
	return ws.closes.Close()
}

func (ws *Workspace) HasLogs() bool {
	return ws.deliveries.HasLogs()
}
