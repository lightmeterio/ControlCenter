// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package driver

import (
	"context"
	"io"
)

type Driver interface {
	ExecuteCommand(ctx context.Context, command []string, stdout, stderr io.Writer) error
}
