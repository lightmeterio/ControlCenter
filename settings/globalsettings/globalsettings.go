// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package globalsettings

import (
	"context"
	"errors"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net"
	"net/url"
)

const (
	SettingKey = "global"
)

var (
	ErrPublicURLInvalid = errors.New("Please provide a full public URL")
	ErrPublicURLNoDNS   = errors.New("Public URL cannot be reached")
)

type IP struct {
	net.IP
}

func (t *IP) MergoFromString(s string) error {
	t.IP = net.ParseIP(s)
	return nil
}

type Settings struct {
	LocalIP     IP     `json:"postfix_public_ip"`
	AppLanguage string `json:"app_language"`
	PublicURL   string `json:"public_url"`
}

type IPAddressGetter interface {
	IPAddress(context.Context) net.IP
}

type MetaReaderGetter struct {
	meta metadata.Reader
}

func New(m metadata.Reader) *MetaReaderGetter {
	return &MetaReaderGetter{meta: m}
}

func (r *MetaReaderGetter) IPAddress(ctx context.Context) net.IP {
	settings, err := GetSettings(ctx, r.meta)

	if err != nil {
		if !errors.Is(err, metadata.ErrNoSuchKey) {
			errorutil.LogErrorf(errorutil.Wrap(err), "obtaining IP address from global settings")
		}

		return nil
	}

	return settings.LocalIP.IP
}

func checkPublicURL(u string) error {
	if len(u) == 0 {
		return nil
	}

	url, err := url.Parse(u)
	if err != nil || len(url.Scheme) == 0 || len(url.Host) == 0 {
		return ErrPublicURLInvalid
	}

	ips, err := net.LookupHost(url.Hostname())
	if err != nil || len(ips) == 0 {
		return ErrPublicURLNoDNS
	}

	return nil
}

func SetSettings(ctx context.Context, writer *metadata.AsyncWriter, settings Settings) error {
	if err := checkPublicURL(settings.PublicURL); err != nil {
		return errorutil.Wrap(err)
	}

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
