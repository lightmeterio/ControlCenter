// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"context"
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/notification"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type creator struct {
	*core.DBCreator
	notifier *notification.Center
}

func newCreator(conn dbconn.RwConn, notifier *notification.Center) (*creator, error) {
	c, err := core.NewCreator(conn)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &creator{DBCreator: c, notifier: notifier}, nil
}

func (c *creator) GenerateInsight(ctx context.Context, tx *sql.Tx, properties core.InsightProperties) error {
	id, err := core.GenerateInsight(ctx, tx, properties)

	if err != nil {
		return errorutil.Wrap(err)
	}

	if err := c.notifier.Notify(notification.Notification{ID: id, Content: properties}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
