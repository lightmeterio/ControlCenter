// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package globalsettings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net"
)

const (
	SettingKey = "global"
)

type IP struct {
	Value string
}

func NewIP(value string) IP {
	return IP{Value: value}
}

func (i *IP) Valid() bool {
	return i.IP() != nil
}

func (i *IP) IP() net.IP {
	return net.ParseIP(i.Value)
}

func (i IP) MarshalJSON() ([]byte, error) {
	buffer := bytes.Buffer{}
	err := json.NewEncoder(&buffer).Encode(i.IP())

	return buffer.Bytes(), err
}

func ipValueOrEmpty(ip net.IP) string {
	if ip != nil {
		return ip.String()
	}

	return ""
}

func (i *IP) UnmarshalJSON(b []byte) error {
	var ip net.IP
	if err := json.NewDecoder(bytes.NewBuffer(b)).Decode(&ip); err != nil {
		return err
	}

	i.Value = ipValueOrEmpty(ip)

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

	return settings.LocalIP.IP()
}

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
