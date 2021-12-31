// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package closers

import (
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
)

// ConvertToCloser is exported to some tests
// nolint:golint
func ConvertToCloser(close func() error) *closer {
	return &closer{CloseFunc: close}
}

type closer struct {
	CloseFunc func() error
}

func (c *closer) Close() error {
	return c.CloseFunc()
}

func New(c ...io.Closer) Closers {
	closers := closers{}

	for _, v := range c {
		closers.Add(v)
	}

	return &closers
}

type Closers interface {
	io.Closer
	Add(io.Closer, ...io.Closer)
}

type closers []io.Closer

func maybeUpdateError(err error, typ io.Closer) error {
	nestedErr := typ.Close()

	if nestedErr == nil {
		return err
	}

	if err != nil {
		return errorutil.BuildChain(nestedErr, err)
	}

	return errorutil.Wrap(nestedErr)
}

func (c *closers) Close() error {
	var err error

	for _, typ := range *c {
		err = maybeUpdateError(err, typ)
	}

	return err
}

func (c *closers) Add(head io.Closer, tail ...io.Closer) {
	*c = append(*c, append([]io.Closer{head}, tail...)...)
}
