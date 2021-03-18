// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package filelogsource

import (
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
)

type Source struct {
	file    io.Reader
	builder transform.Builder
}

func New(file io.Reader, builder transform.Builder) (*Source, error) {
	return &Source{
		file:    file,
		builder: builder,
	}, nil
}

func (s *Source) PublishLogs(p postfix.Publisher) error {
	if err := transform.ReadFromReader(s.file, p, s.builder); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
