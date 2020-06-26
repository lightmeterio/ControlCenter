package workspace

import (
	"os"
	"time"

	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/logdb"
	"gitlab.com/lightmeter/controlcenter/util"
)

type Workspace struct {
	config data.Config
	logs   logdb.DB
	auth   *auth.Auth
}

func NewWorkspace(workspaceDirectory string, config data.Config) (Workspace, error) {
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

	return Workspace{
		config: config,
		logs:   logDb,
		auth:   auth,
	}, nil
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

func (ws *Workspace) Run() <-chan interface{} {
	return ws.logs.Run()
}

func (ws *Workspace) Close() error {
	return ws.logs.Close()
}

func (ws *Workspace) HasLogs() bool {
	return ws.logs.HasLogs()
}
