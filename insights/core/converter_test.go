// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core_test

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	notificationCore "gitlab.com/lightmeter/controlcenter/notification/core"
	"sync"
	"testing"
)

type fakeContent struct {
	B string `json:"b"`
}

func (c fakeContent) Title() notificationCore.ContentComponent {
	return fakeComponent{}
}

func (c fakeContent) Description() notificationCore.ContentComponent {
	return fakeComponent{}
}

func (c fakeContent) Metadata() notificationCore.ContentMetadata {
	return nil
}

type fakeComponent struct{}

func (c fakeComponent) String() string {
	return ""
}

func (c fakeComponent) TplString() string {
	return ""
}

func (c fakeComponent) Args() []interface{} {
	return nil
}

func TestDefaultContentTypeDecoder(t *testing.T) {
	Convey("DefaultContentTypeDecoder", t, func(c C) {
		var wg sync.WaitGroup

		decode := core.DefaultContentTypeDecoder(&fakeContent{})

		json := []byte(`{"b": "test"}`)

		for i := 1; i < 100; i++ {
			wg.Add(1)
			go func() {
				content, err := decode(json)
				c.So(err, ShouldBeNil)
				c.So(content.(*fakeContent).B, ShouldEqual, "test")
				wg.Done()
			}()
		}

		wg.Wait()
	})
}
