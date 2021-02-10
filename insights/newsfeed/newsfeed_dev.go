// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build dev !release

package newsfeed

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

// Executed only on development builds, for better developer experience
func (d *detector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	if err := generateInsight(tx, c, d.creator, Content{
		Title:       "Sample Newsfeed insight!!!",
		Description: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Vestibulum vitae nunc lacinia, fermentum nisi vitae, iaculis est. In ut mi libero. Nullam ac varius velit, sed ullamcorper ipsum. Nulla sollicitudin, purus vel commodo venenatis, urna eros auctor odio, vel viverra arcu dui vel leo. Aenean mattis ut erat ac tempor.",
		Link:        "https://example.com/posts/42",
		Published:   time.Date(2020, time.February, 22, 1, 2, 3, 0, time.UTC),
		GUID:        "https://example.com/?p=42",
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
