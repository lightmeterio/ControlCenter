// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

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

func TestCloseAllAdd(t *testing.T) {
	Convey("CloseAll", t, func() {
		closers := closers{}
		close := ConvertToCloser(func() error {
			return errorutil.Wrap(errors.New("closes 3"))
		})
		closers.Add(close)

		close = ConvertToCloser(func() error {
			return errorutil.Wrap(errors.New("closes 1"))
		})
		closers.Add(close)

		close = ConvertToCloser(func() error {
			return errorutil.Wrap(errors.New("closes 2"))
		})
		closers.Add(close)

		So(len(closers), ShouldEqual, 3)

		Convey("close return errors", func() {
			So(closers.Close(), ShouldNotBeNil)
		})
	})
}
