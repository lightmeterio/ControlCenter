// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type fetcher struct {
	core.Fetcher
}

func newFetcher(pool *dbconn.RoPool) (*fetcher, error) {
	f, err := core.NewFetcher(pool)

	if err != nil {
		return nil, errorutil.Wrap(err)
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
		return nil, errorutil.Wrap(err)
	}

	return &creator{DBCreator: c, notifier: notifier}, nil
}

func (c *creator) GenerateInsight(tx *sql.Tx, properties core.InsightProperties) error {
	id, err := core.GenerateInsight(tx, properties)

	if err != nil {
		return errorutil.Wrap(err)
	}

	if properties.Rating == core.BadRating {
		if err := c.notifier.Notify(notification.Notification{ID: id, Content: properties.Content}); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}
