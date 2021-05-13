// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build dev !release

package detectiveescalation

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

// Executed only on development builds, for better developer experience
func (d *detector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	if err := generateInsight(tx, c, d.creator, SampleInsightContent); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
