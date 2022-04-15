// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package walkthrough

import (
	"context"

	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/settingsutil"
)

type Settings struct {
	Completed bool `json:"completed"`
}

const SettingsKey = "walkthrough"

func SetSettings(ctx context.Context, writer *metadata.AsyncWriter, settings Settings) error {
	return settingsutil.Set[Settings](ctx, writer, settings, SettingsKey)
}

func GetSettings(ctx context.Context, reader metadata.Reader) (*Settings, error) {
	return settingsutil.Get[Settings](ctx, reader, SettingsKey)
}
