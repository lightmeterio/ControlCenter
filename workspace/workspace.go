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
	"log"
	"os"
	"path"
	"time"
)

type Workspace struct {
	runner.CancelableRunner

	logs           *deliverydb.DB
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

	logDb, err := deliverydb.New(workspaceDirectory)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	tracker, err := tracking.New(workspaceDirectory, logDb.ResultsPublisher())
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

	dashboard, err := dashboard.New(logDb.ReadConnection())

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

	ws := &Workspace{
		logs:                logDb,
		tracker:             tracker,
		insightsEngine:      insightsEngine,
		auth:                auth,
		rblDetector:         rblDetector,
		rblChecker:          rblChecker,
		settingsMetaHandler: m,
		settingsRunner:      settingsRunner,
		closes: closeutil.New(
			auth,
			&logDb,
			insightsEngine,
			m,
		),
		NotificationCenter: notificationCenter,
	}

	return w, nil
}

func (ws *Workspace) SettingsAcessors() (*meta.AsyncWriter, *meta.Reader) {
	return ws.settingsRunner.Writer(), ws.settingsMetaHandler.Reader
}

func (ws *Workspace) InsightsFetcher() insightsCore.Fetcher {
	return ws.insightsEngine.Fetcher()
}

func (ws *Workspace) Dashboard() (dashboard.Dashboard, error) {
	return dashboard.New(ws.logs.ReadConnection())
}

func (ws *Workspace) Auth() *auth.Auth {
	return ws.auth
}

// Obtain the most recent time inserted in the database,
// or a zero'd time in case case no value has been found
func (ws *Workspace) MostRecentLogTime() time.Time {
	return ws.logs.MostRecentLogTime()
}

func (ws *Workspace) NewPublisher() data.Publisher {
	return data.ComposedPublisher{ws.logs.NewPublisher(), ws.rblDetector.NewPublisher()}
}

func (ws *Workspace) Run() <-chan struct{} {
	ws.rblChecker.StartListening()

	doneInsights, cancelInsights := ws.insightsEngine.Run()
	doneSettings, cancelSettings := ws.settingsRunner.Run()
	doneMsgRbl, cancelMsgRbl := ws.rblDetector.Run()

	done := make(chan struct{})

	go func() {
		// NOTE: for now the workspace execution can be stoped by simply stopping
		// feeding it with log lines, by closing the log publisher.
		// TODO: this is a very unclear operation mode and needs to be changed
		// or better documented
		<-ws.logs.Run()

		cancelInsights()
		cancelSettings()
		cancelMsgRbl()

		doneInsights()
		doneSettings()
		doneMsgRbl()

		done <- struct{}{}
	}()

	return done
}

func (ws *Workspace) Close() error {
	return ws.closes.Close()
}

func (ws *Workspace) HasLogs() bool {
	return ws.logs.HasLogs()
}
