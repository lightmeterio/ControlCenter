package workspace

import (
	"os"
	"path"
	"time"

	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights"
	insightsCore "gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/logdb"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/newsletter"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/settings"
	"gitlab.com/lightmeter/controlcenter/util"
)

type Workspace struct {
	logs           logdb.DB
	insightsEngine *insights.Engine
	auth           *auth.Auth

	metaConnPair dbconn.ConnPair
	meta         *meta.MetadataHandler
	settings     *settings.MasterConf
}

func NewWorkspace(workspaceDirectory string, config logdb.Config) (Workspace, error) {
	if err := os.MkdirAll(workspaceDirectory, os.ModePerm); err != nil {
		return Workspace{}, util.WrapError(err, "Error creating working directory ", workspaceDirectory)
	}

	logDb, err := logdb.Open(workspaceDirectory, config)

	if err != nil {
		return Workspace{}, util.WrapError(err)
	}

	auth, err := auth.NewAuth(workspaceDirectory, auth.Options{})

	if err != nil {
		return Workspace{}, util.WrapError(err)
	}

	metadataConnPair, err := dbconn.NewConnPair(path.Join(workspaceDirectory, "master.db"))

	if err != nil {
		return Workspace{}, util.WrapError(err)
	}

	m, err := meta.NewMetaDataHandler(metadataConnPair, "master")

	if err != nil {
		return Workspace{}, util.WrapError(err)
	}

	settings, err := settings.NewMasterConf(m, newsletter.NewSubscriber("https://phplist.lightmeter.io/"))

	if err != nil {
		return Workspace{}, util.WrapError(err)
	}

	dashboard, err := dashboard.New(logDb.ReadConnection())

	if err != nil {
		return Workspace{}, util.WrapError(err)
	}

	// TODO: use an actual notification center!
	notificationCenter := &dummyNotificationCenter{}

	insightsEngine, err := insights.NewEngine(workspaceDirectory, notificationCenter, insightsOptions(dashboard))

	if err != nil {
		return Workspace{}, util.WrapError(err)
	}

	return Workspace{
		logs:           logDb,
		insightsEngine: insightsEngine,
		auth:           auth,
		metaConnPair:   metadataConnPair,
		meta:           m,
		settings:       settings,
	}, nil
}

func (ws *Workspace) InsightsFetcher() insightsCore.Fetcher {
	return ws.insightsEngine.Fetcher()
}

func (ws *Workspace) Settings() *settings.MasterConf {
	return ws.settings
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

	done := make(chan struct{})

	go func() {
		<-ws.logs.Run()
		cancelInsights()
		doneInsights()
		done <- struct{}{}
	}()

	return done
}

func (ws *Workspace) Close() error {
	if err := ws.settings.Close(); err != nil {
		return util.WrapError(err)
	}

	if err := ws.meta.Close(); err != nil {
		return util.WrapError(err)
	}

	if err := ws.metaConnPair.Close(); err != nil {
		return util.WrapError(err)
	}

	if err := ws.auth.Close(); err != nil {
		return util.WrapError(err)
	}

	if err := ws.logs.Close(); err != nil {
		return util.WrapError(err)
	}

	if err := ws.insightsEngine.Close(); err != nil {
		return util.WrapError(err)
	}

	return nil
}

func (ws *Workspace) HasLogs() bool {
	return ws.logs.HasLogs()
}

type dummyNotificationCenter struct {
}

func (*dummyNotificationCenter) Notify(notification.Content) {
	// implement notification.Center
}
