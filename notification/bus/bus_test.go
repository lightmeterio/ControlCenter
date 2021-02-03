// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package bus_test

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/notification/bus"
	"sync/atomic"
	"testing"
)

func TestBus(t *testing.T) {

	Convey("Bus", t, func() {

		bus := bus.New()

		Convey("Flow", func() {

			var counter int32
			bus.AddEventListener("kind1", func(msg string) error {
				t.Log("AddEventListener")
				t.Log(msg)
				atomic.AddInt32(&counter, 1)
				return nil
			})

			bus.AddEventListener("kind2", func(msg string) error {
				t.Log("AddEventListener")
				t.Log(msg)
				atomic.AddInt32(&counter, 1)
				return nil
			})

			err := bus.Publish("Hello world")
			So(err, ShouldBeNil)
			So(atomic.LoadInt32(&counter), ShouldEqual, 2)

			// Reset counter
			atomic.AddInt32(&counter, -2)

			bus.UpdateEventListener("kind1", func(msg string) error {
				t.Log("UpdateEventListener")
				t.Log(msg)
				atomic.AddInt32(&counter, 1)
				return nil
			})
			err = bus.Publish("Hello world")
			So(err, ShouldBeNil)

			So(atomic.LoadInt32(&counter), ShouldEqual, 2)

		})
	})
}

func TestBusPanic(t *testing.T) {

	Convey("Bus", t, func() {

		bus := bus.New()

		Convey("Flow panic", func() {
			So(func() {
				bus.AddEventListener("kind1", func(msg string) error {
					t.Log("AddEventListener")
					t.Log(msg)
					return nil
				})

				bus.AddEventListener("kind1", func(msg string) error {
					t.Log("AddEventListener")
					t.Log(msg)
					return nil
				})
			}, ShouldPanic)
		})
	})
}
