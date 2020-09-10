package insights

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/util"
)

type fetcher struct {
	core.Fetcher
}

func newFetcher(conn dbconn.RoConn) (*fetcher, error) {
	f, err := core.NewFetcher(conn)

	if err != nil {
		return nil, util.WrapError(err)
	}

	return &fetcher{Fetcher: f}, nil
}

type creator struct {
	*core.DBCreator
	notifier notification.Center
}

func newCreator(conn dbconn.RwConn, notifier notification.Center) (*creator, error) {
	c, err := core.NewCreator(conn)

	if err != nil {
		return nil, util.WrapError(err)
	}

	return &creator{DBCreator: c, notifier: notifier}, nil
}

type InsightNotification struct {
	ID int64
}

func (c *creator) GenerateInsight(tx *sql.Tx, properties core.InsightProperties) error {
	id, err := core.GenerateInsight(tx, properties)

	if err != nil {
		return util.WrapError(err)
	}

	c.notifier.Notify(InsightNotification{ID: id})

	return nil
}
