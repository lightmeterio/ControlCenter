// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package featureflags

import (
	"context"
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/metadata"
)

type fakeReader struct {
	v *Settings
}

func (r *fakeReader) Retrieve(context.Context, metadata.Key) (metadata.Value, error) {
	// FIMXE: refused bequest?
	panic("Not Implemented")
}

func (r *fakeReader) RetrieveJson(ctx context.Context, key metadata.Key, value metadata.Value) error {
	if r.v == nil {
		return metadata.ErrNoSuchKey
	}

	*(value.(*Settings)) = *r.v

	return nil
}

func TestFeatures(t *testing.T) {
	Convey("Test Features", t, func() {
		Convey("No value", func() {
			reader := &fakeReader{}
			_, err := GetSettings(context.Background(), reader)
			So(errors.Is(err, metadata.ErrNoSuchKey), ShouldBeTrue)
		})

		Convey("Empty Value", func() {
			reader := &fakeReader{v: &Settings{}}
			s, err := GetSettings(context.Background(), reader)
			So(err, ShouldBeNil)
			So(s, ShouldNotBeNil)
			So(s, ShouldResemble, &Settings{})
		})

		Convey("Change single key", func() {
			reader := &fakeReader{v: &Settings{DisableInsightsView: true}}
			s, err := GetSettings(context.Background(), reader)
			So(err, ShouldBeNil)
			So(s, ShouldNotBeNil)
			So(s, ShouldResemble, &Settings{DisableInsightsView: true})
		})

		Convey("Simple view is a catch-all for other flags", func() {
			reader := &fakeReader{v: &Settings{EnableSimpleView: true}}
			s, err := GetSettings(context.Background(), reader)
			So(err, ShouldBeNil)
			So(s, ShouldNotBeNil)

			policyLink := "https://lightmeter.io/privacy-policy-delivery/"
			projectLink := "https://getlightmeter.com/"

			So(s, ShouldResemble, &Settings{
				DisableInsightsView:        true,
				DisableV1Dashboard:         true,
				EnableV2Dashboard:          true,
				DisableRawLogs:             true,
				EnableSimpleView:           true,
				AlternativePolicyLink:      &policyLink,
				AlternativeProjectMainLink: &projectLink,
			})
		})
	})
}
