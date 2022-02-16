// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package insights

import (
	"context"
	"errors"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type Settings struct {
	// High bounce rate insight
	BounceRateThreshold int `json:"bounce_rate_threshold"`

	// Mail inactivity settings
	MailInactivityLookupRange int `json:"mail_inactivity_lookup_range"`
	MailInactivityMinInterval int `json:"mail_inactivity_min_interval"`
}

const SettingKey = "insights"

func SetSettings(ctx context.Context, writer *metadata.AsyncWriter, settings Settings) error {
	if err := writer.StoreJsonSync(ctx, SettingKey, settings); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func GetSettings(ctx context.Context, reader metadata.Reader) (*Settings, error) {
	var settings Settings

	err := reader.RetrieveJson(ctx, SettingKey, &settings)
	if err != nil && errors.Is(err, metadata.ErrNoSuchKey) {
		return &Settings{
			// default settings
			BounceRateThreshold:       5,
			MailInactivityLookupRange: 24,
			MailInactivityMinInterval: 12,
		}, nil
	}

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &settings, nil
}
