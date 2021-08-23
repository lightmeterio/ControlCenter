// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build dev !release

package messagerblinsight

import (
	"context"
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

// Executed only on development builds, for better developer experience
func (d *detector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	if err := generateInsight(tx, c.Now(), d.creator, Content{
		Address:   globalsettings.IPAddress(context.Background()),
		Message:   "Sample Insight: host blocked. Try https://google.com/ to unblock it",
		Recipient: "some.mail.com",
		Status:    parser.DeferredStatus.String(),
		Host:      "Google",
		Time:      c.Now(),
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
