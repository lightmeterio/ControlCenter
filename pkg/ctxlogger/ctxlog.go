// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package ctxlogger

import (
	"context"
	"github.com/rs/zerolog"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

const LoggerKey string = "LoggerKey"

func GetCtxLogger(ctx context.Context) *zerolog.Logger {
	return ctx.Value(LoggerKey).(*zerolog.Logger)
}

func LogErrorf(ctx context.Context, err error, format string, args ...interface{}) {
	GetCtxLogger(ctx).Error().Interface("error", errorutil.ExpandError(err)).Msgf(format, args...)
}
