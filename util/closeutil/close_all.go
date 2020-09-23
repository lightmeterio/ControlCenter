package closeutil

import (
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
)

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

func (c *Closers) Close() error {
	if len(*c) == 0 {
		panic("close funcs are missing")
	}

	var err error
	for _, typ := range *c {
		if typ == nil {
			panic("closer is nil")
		}
		err = func() error {
			nestedErr := typ.Close()

			if nestedErr == nil {
				return err
			}

			if err == nil {
				return nestedErr
			}

			return errorutil.BuildChain(nestedErr, err)
		}()
	}
	return err
}
