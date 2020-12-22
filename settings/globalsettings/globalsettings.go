package globalsettings

import (
	"context"
	"errors"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net"
)

const (
	SettingsKey = "global"
)

type Settings struct {
	LocalIP     net.IP `json:"local_ip"`
	APPLanguage string `json:"app_language"`
}

type AppLanguageGetter interface {
	AppLanguage(ctx context.Context) string
}

type IPAddressGetter interface {
	IPAddress(context.Context) net.IP
}

type Getter interface {
	IPAddressGetter
	AppLanguageGetter
}

type MetaReaderGetter struct {
	meta *meta.Reader
}

func New(m *meta.Reader) *MetaReaderGetter {
	return &MetaReaderGetter{meta: m}
}

func (r *MetaReaderGetter) IPAddress(ctx context.Context) net.IP {
	var settings Settings
	err := r.meta.RetrieveJson(ctx, SettingsKey, &settings)

	if err != nil {
		if !errors.Is(err, meta.ErrNoSuchKey) {
			errorutil.LogErrorf(errorutil.Wrap(err), "obtaining IP address from global settings")
		}

		return nil
	}

	return settings.LocalIP
}

func (r *MetaReaderGetter) AppLanguage(ctx context.Context) string {
	var settings Settings
	err := r.meta.RetrieveJson(ctx, SettingsKey, &settings)

	if err != nil {
		if !errors.Is(err, meta.ErrNoSuchKey) {
			errorutil.LogErrorf(errorutil.Wrap(err), "obtaining APP language from global settings")
		}

		return ""
	}

	return settings.APPLanguage
}
