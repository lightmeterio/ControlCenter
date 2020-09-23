package closeutil

import (
	"errors"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCloseAll(t *testing.T) {
	Convey("CloseAll", t, func() {
		closers := []io.Closer{
			ConvertToCloser(func() error {
				return errorutil.Wrap(errors.New("closes 1"))
			}),
			ConvertToCloser(func() error {
				return errorutil.Wrap(errors.New("closes 2"))
			}),
			ConvertToCloser(func() error {
				return errorutil.Wrap(errors.New("closes 3"))
			}),
		}
		Convey("close return errors", func() {
			closers := New(closers...)
			ShouldNotBeNil(closers.Close())
		})
	})
}