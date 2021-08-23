// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package globalsettings

import (
	"context"
	"errors"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net"
)

const (
	SettingKey = "global"
)

type Settings struct {
	LocalIP     net.IP `json:"postfix_public_ip"`
	APPLanguage string `json:"app_language"`
	PublicURL   string `json:"public_url"`
}

func IPAddress(ctx context.Context) net.IP {
	settings, err := GetSettings(ctx)

	if err != nil {
		if !errors.Is(err, meta.ErrNoSuchKey) {
			errorutil.LogErrorf(errorutil.Wrap(err), "obtaining IP address from global settings")
		}

		return nil
	}

	return settings.LocalIP
}

func SetSettings(ctx context.Context, writer *meta.AsyncWriter, settings Settings) error {
	if err := writer.StoreJsonSync(ctx, SettingKey, settings); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func GetSettings(ctx context.Context) (*Settings, error) {
	var settings Settings

	err := meta.RetrieveJson(ctx, dbconn.DbMaster, SettingKey, &settings)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &settings, nil
}
