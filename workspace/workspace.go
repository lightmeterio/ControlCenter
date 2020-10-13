package workspace

import (
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights"
	insightsCore "gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/logdb"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"os"
	"path"
	"time"
)

type Workspace struct {
	logs           *logdb.DB
	insightsEngine *insights.Engine
	auth           *auth.Auth

	NotificationCenter notification.Center

	settingsMetaHandler *meta.Handler
	settingsRunner      *meta.Runner

	closes closeutil.Closers
}

func NewWorkspace(workspaceDirectory string, config logdb.Config) (Workspace, error) {
	if err := os.MkdirAll(workspaceDirectory, os.ModePerm); err != nil {
		return Workspace{}, errorutil.Wrap(err, "Error creating working directory ", workspaceDirectory)
	}

	logDb, err := logdb.Open(workspaceDirectory, config)

	if err != nil {
		return Workspace{}, errorutil.Wrap(err)
	}

	auth, err := auth.NewAuth(workspaceDirectory, auth.Options{})

	if err != nil {
		return Workspace{}, errorutil.Wrap(err)
	}

	metadataConnPair, err := dbconn.NewConnPair(path.Join(workspaceDirectory, "master.db"))

	if err != nil {
		return Workspace{}, errorutil.Wrap(err)
	}

	m, err := meta.NewHandler(metadataConnPair, "master")

	if err != nil {
		return Workspace{}, errorutil.Wrap(err)
	}

	settingsRunner := meta.NewRunner(m)

	if err != nil {
		return Workspace{}, errorutil.Wrap(err)
	}

	dashboard, err := dashboard.New(logDb.ReadConnection())

	if err != nil {
		return Workspace{}, errorutil.Wrap(err)
	}

	notificationCenter := notification.New(m.Reader)

	rblChecker := localrbl.NewChecker(m.Reader, localrbl.Options{
		NumberOfWorkers:  10,
		Lookup:           localrbl.RealLookup,
		RBLProvidersURLs: localrbl.DefaultRBLs,
	})

	// FIXME: rblChecker should start on Run()!!!
	rblChecker.StartListening()

	insightsEngine, err := insights.NewEngine(
		workspaceDirectory,
		notificationCenter, insightsOptions(dashboard, rblChecker))

	if err != nil {
		return Workspace{}, errorutil.Wrap(err)
	}

	// closes can be mocked or stubbed out
	w := Workspace{
		logs:                &logDb,
		insightsEngine:      insightsEngine,
		auth:                auth,
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
	return ws.logs.NewPublisher()
}

func (ws *Workspace) Run() <-chan struct{} {
	doneInsights, cancelInsights := ws.insightsEngine.Run()
	doneSettings, cancelSettings := ws.settingsRunner.Run()

	done := make(chan struct{})

	go func() {
		// NOTE: for now the workspace execution can be stoped by simply stopping
		// feeding it with log lines, by closing the log publisher.
		// TODO: this is a very unclear operation mode and needs to be changed
		// or better documented
		<-ws.logs.Run()

		cancelInsights()
		cancelSettings()

		doneInsights()
		doneSettings()

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
