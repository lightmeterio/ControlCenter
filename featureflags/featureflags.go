// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package featureflags

import (
	"context"

	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/settingsutil"
)

type Settings struct {
	DisableV1Dashboard         bool    `json:"disable_v1_dashboard"`
	EnableV2Dashboard          bool    `json:"enable_v2_dashboard"`
	DisableInsightsView        bool    `json:"disable_insights_view"`
	DisableRawLogs             bool    `json:"disable_raw_logs"`
	EnableSimpleView           bool    `json:"enable_simple_view"`
	AlternativePolicyLink      *string `json:"policy_link,omitempty"`
	AlternativeProjectMainLink *string `json:"project_link,omitempty"`
}

const SettingsKey = `feature_flags`

var AlternativePolicyLink = "https://lightmeter.io/privacy-policy-delivery/"
var AlternativeProjectmainLink = "https://getlightmeter.com/"

func GetSettings(ctx context.Context, reader metadata.Reader) (*Settings, error) {
	s, err := settingsutil.Get[Settings](ctx, reader, SettingsKey)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	if s.EnableSimpleView {
		s.DisableV1Dashboard = true
		s.EnableV2Dashboard = true
		s.DisableInsightsView = true
		s.DisableRawLogs = true
		s.AlternativePolicyLink = &AlternativePolicyLink
		s.AlternativeProjectMainLink = &AlternativeProjectmainLink
	}

	return s, nil
}
