// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package featureflags

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type Settings struct {
	EnableV2Dashboard bool `json:"enable_v2_dashboard"`
}

var SettingsKey = `feature_flags`

// FIXME: this is copie&pasted from settings and other many places over the codebase
// NOTE: this is a good candidate for using Go generics
func GetSettings(ctx context.Context, reader metadata.Reader) (*Settings, error) {
	var settings Settings

	err := reader.RetrieveJson(ctx, SettingsKey, &settings)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &settings, nil
}
