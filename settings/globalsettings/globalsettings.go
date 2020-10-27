package globalsettings

import (
	"context"
	"errors"
	"gitlab.com/lightmeter/controlcenter/meta"
	"log"
	"net"
)

const (
	SettingsKey = "global"
)

type Settings struct {
	LocalIP net.IP `json:"local_ip"`
}

type Getter interface {
	IPAddress(context.Context) net.IP
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
			log.Printf("Error obtaining IP address from global settings: %v", err)
		}

		return nil
	}

	return settings.LocalIP
}
