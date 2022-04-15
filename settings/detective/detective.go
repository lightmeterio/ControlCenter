// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package detective

import (
	"context"

	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/settingsutil"
)

type Settings struct {
	EndUsersEnabled bool `json:"end_users_enabled"`
}

const SettingKey = "detective"

func SetSettings(ctx context.Context, writer *metadata.AsyncWriter, settings Settings) error {
	return settingsutil.Set[Settings](ctx, writer, settings, SettingKey)
}

func GetSettings(ctx context.Context, reader metadata.Reader) (*Settings, error) {
	return settingsutil.Get[Settings](ctx, reader, SettingKey)
}
