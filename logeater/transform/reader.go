// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package transform

import (
	"bufio"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
)

func ReadFromReader(reader io.Reader, pub postfix.Publisher, build Builder) error {
	scanner := bufio.NewScanner(reader)

	t, err := build()
	if err != nil {
		return errorutil.Wrap(err)
	}

	for {
		if !scanner.Scan() {
			break
		}

		r, err := t.Transform(scanner.Bytes())
		if err != nil {
			log.Err(err).Msgf("Error reading from reader: %v", reader)
		}

		pub.Publish(r)
	}

	return nil
}
