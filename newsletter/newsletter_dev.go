// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build dev !release

package newsletter

import (
	"context"
	"github.com/rs/zerolog/log"
)

type dummySubscriber struct{}

func (*dummySubscriber) Subscribe(context context.Context, email string) error {
	log.Info().Msgf("A dummy call that would otherwise subscribe email %v to Lightmeter newsletter :-)", email)
	return nil
}

func NewSubscriber(string) Subscriber {
	return &dummySubscriber{}
}
