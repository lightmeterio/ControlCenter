// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package globalsettings

import (
	"context"
	"errors"
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

type IPAddressGetter interface {
	IPAddress(context.Context) net.IP
}

type MetaReaderGetter struct {
	meta *meta.Reader
}

func New(m *meta.Reader) *MetaReaderGetter {
	return &MetaReaderGetter{meta: m}
}

func (r *MetaReaderGetter) IPAddress(ctx context.Context) net.IP {
	settings, err := GetSettings(ctx, r.meta)

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

func GetSettings(ctx context.Context, reader *meta.Reader) (*Settings, error) {
	var settings Settings

	err := reader.RetrieveJson(ctx, SettingKey, &settings)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &settings, nil
}
