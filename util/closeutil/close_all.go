// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package closeutil

import (
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
)

// ConvertToCloser is exported to some tests
// nolint:golint
func ConvertToCloser(close func() error) *closer {
	if close == nil {
		panic("close is nil")
	}

	return &closer{CloseFunc: close}
}

type closer struct {
	CloseFunc func() error
}

func (c *closer) Close() error {
	return c.CloseFunc()
}

func New(closers ...io.Closer) Closers {
	return closers
}

type Closers []io.Closer

func maybeUpdateError(err error, typ io.Closer) error {
	nestedErr := typ.Close()

	if nestedErr == nil {
		return err
	}

	if err == nil {
		return errorutil.Wrap(nestedErr)
	}

	return errorutil.BuildChain(nestedErr, err)
}

func (c *Closers) Close() error {
	var err error

	for _, typ := range *c {
		if typ == nil {
			panic("closer is nil")
		}

		err = maybeUpdateError(err, typ)
	}

	return err
}

func (c *Closers) Add(cs ...io.Closer) {
	if len(cs) == 0 {
		panic("close funcs are missing")
	}

	*c = append(*c, cs...)
}
