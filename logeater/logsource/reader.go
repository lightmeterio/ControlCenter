// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package logsource

import (
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type Reader struct {
	source Source
	pub    data.Publisher
}

func NewReader(source Source, pub data.Publisher) Reader {
	return Reader{source: source, pub: pub}
}

func (r *Reader) Run() error {
	if err := r.source.PublishLogs(r.pub); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
