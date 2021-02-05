// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package core_test

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"sync"
	"testing"
)

type fakeContent struct {
	B string `json:"b"`
}

func (c fakeContent) String() string { return ""}

func (c fakeContent) TplString() string { return "" }

func (c fakeContent) Args() []interface{} { return []interface{}{} }

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
