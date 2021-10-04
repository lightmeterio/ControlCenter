// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package collector

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
	"time"
)

type Reporter interface {
	io.Closer
	ExecutionInterval() time.Duration
	Step(tx *sql.Tx, clock timeutil.Clock) error
	ID() string
}
