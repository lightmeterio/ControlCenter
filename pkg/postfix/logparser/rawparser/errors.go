// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

import "errors"

var (
	ErrInvalidHeaderLine  = errors.New("Could not parse header")
	ErrUnsupportedLogLine = errors.New("Unsupported payload")
)
