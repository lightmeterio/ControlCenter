// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package server

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
)

type fakeReader struct {
	s *globalsettings.Settings
}

func (r *fakeReader) Retrieve(context.Context, metadata.Key) (metadata.Value, error) {
	// FIMXE: refused bequest?
	panic("Not Implemented")
}

func (r *fakeReader) RetrieveJson(ctx context.Context, key metadata.Key, value metadata.Value) error {
	if r.s == nil {
		return metadata.ErrNoSuchKey
	}

	*(value.(*globalsettings.Settings)) = *r.s

	return nil
}

func TestPublicURL(t *testing.T) {
	Convey("Public URL", t, func() {
		Convey("No settings, no URL", func() {
			reader := &fakeReader{s: nil}
			url, err := getPublicURL(context.Background(), reader)
			So(err, ShouldBeNil)
			So(url, ShouldBeNil)
		})

		Convey("Empty URL", func() {
			reader := &fakeReader{s: &globalsettings.Settings{PublicURL: ""}}
			url, err := getPublicURL(context.Background(), reader)
			So(err, ShouldBeNil)
			So(url, ShouldBeNil)
		})

		Convey("Valid URL", func() {
			reader := &fakeReader{s: &globalsettings.Settings{PublicURL: "https://example.com/some/funny/path/#settings"}}
			url, err := getPublicURL(context.Background(), reader)
			So(err, ShouldBeNil)
			So(url, ShouldNotBeNil)
			So(url.Hostname(), ShouldEqual, "example.com")
			So(url.Path, ShouldEqual, "/some/funny/path/")
		})
	})
}
