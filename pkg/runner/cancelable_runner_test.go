package runner

import (
	. "github.com/smartystreets/goconvey/convey"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCancelableRunner(t *testing.T) {
	Convey("cancelable runner", t, func() {

		Convey("run", func() {
			var counter int32
			execute := func(done DoneChan, cancel CancelChan) {
				go func() {
					done <- struct{}{}
					atomic.AddInt32(&counter, 1)
				}()
			}

			runner := NewCancelableRunner(execute)
		    done, _ := runner.Run()
		    done()
			if atomic.LoadInt32(&counter) != 1 {
				So(t, ShouldEqual, 1)
			}
		})
	})
}

func TestCancelableRunnerCancel(t *testing.T) {
	Convey("cancelable runner", t, func() {

		Convey("run", func() {
			var wg sync.WaitGroup
			var counter int32
			wg.Add(1)
			execute := func(done DoneChan, cancel CancelChan) {
				go func() {
					time.After(time.Second * 1 / 2)
					<-cancel
					atomic.AddInt32(&counter, 1)
					wg.Done()
				}()
			}

			runner := NewCancelableRunner(execute)
			_, cancel := runner.Run()
			cancel()
			wg.Wait()
			if atomic.LoadInt32(&counter) != 1 {
				So(t, ShouldEqual, 1)
			}
		})
	})
}