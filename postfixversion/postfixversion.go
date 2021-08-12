// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postfixversion

import (
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
)

const SettingKey = "postfix_version"

type Publisher struct {
	settingsWriter *meta.AsyncWriter
}

func (p Publisher) Publish(r postfix.Record) {
	if version, ok := r.Payload.(parser.Version); ok {
		result := p.settingsWriter.StoreJson(SettingKey, version)

		go func() {
			if err := <-result.Done(); err != nil {
				log.Err(err).Msgf("Could not store postfix version in database: %v", err)
			}
		}()
	}
}

func NewPublisher(settingsWriter *meta.AsyncWriter) Publisher {
	return Publisher{settingsWriter}
}
