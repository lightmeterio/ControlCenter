package workspace

import (
	"errors"
	"os"

	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/logdb"
)

type Workspace struct {
	config data.Config
	logs   logdb.DB
}

func NewWorkspace(workspaceDirectory string, config data.Config) (Workspace, error) {
	if err := os.MkdirAll(workspaceDirectory, os.ModePerm); err != nil {
		return Workspace{}, errors.New("Error creating working directory " + workspaceDirectory + ": " + err.Error())
	}

	logDb, err := logdb.Open(workspaceDirectory, config)

	if err != nil {
		return Workspace{}, err
	}

	return Workspace{
		config: config,
		logs:   logDb,
	}, nil
}

func (ws *Workspace) Dashboard() (dashboard.Dashboard, error) {
	return dashboard.New(ws.logs.ReadConnection())
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
