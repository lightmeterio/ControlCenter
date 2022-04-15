// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package featureflags

import (
	"context"

	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/settingsutil"
)

type Settings struct {
	DisableV1Dashboard  bool `json:"disable_v1_dashboard"`
	EnableV2Dashboard   bool `json:"enable_v2_dashboard"`
	DisableInsightsView bool `json:"disable_insights_view"`
	DisableRawLogs      bool `json:"disable_raw_logs"`
}

const SettingsKey = `feature_flags`

func GetSettings(ctx context.Context, reader metadata.Reader) (*Settings, error) {
	return settingsutil.Get[Settings](ctx, reader, SettingsKey)
}
