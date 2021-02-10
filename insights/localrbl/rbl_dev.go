// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build dev !release

package localrblinsight

import (
	"context"
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

// Executed only on development builds, for better developer experience
func (d *detector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	if err := generateInsight(tx, c, d.creator, content{
		ScanInterval: data.TimeInterval{From: c.Now(), To: c.Now().Add(time.Second * 30)},
		Address:      d.options.Checker.IPAddress(context.Background()),
		RBLs: []localrbl.ContentElement{
			{RBL: "rbl.com", Text: "Funny reason"},
			{RBL: "anotherrbl.de", Text: "Another funny reason at http://example.com Website"},
		},
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
