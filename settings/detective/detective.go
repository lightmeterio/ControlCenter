// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package detective

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type Settings struct {
	EndUsersEnabled bool `json:"end_users_enabled"`
}

const SettingKey = "detective"

func SetSettings(ctx context.Context, writer *metadata.AsyncWriter, settings Settings) error {
	if err := writer.StoreJsonSync(ctx, SettingKey, settings); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func GetSettings(ctx context.Context, reader metadata.Reader) (*Settings, error) {
	var settings Settings

	err := reader.RetrieveJson(ctx, SettingKey, &settings)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &settings, nil
}
