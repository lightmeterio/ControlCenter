package localrbl

import (
	"github.com/mrichman/godnsbl"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"net"
	"strings"
	"testing"
	"time"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

func TestDnsRBL(t *testing.T) {
	Convey("Test Local RBL", t, func() {
		lookup := func(rblList string, targetHost string) godnsbl.RBLResults {
			time.Sleep(200 * time.Millisecond)

			if !strings.HasSuffix(rblList, "-blocked") {
				return godnsbl.RBLResults{}
			}

			return godnsbl.RBLResults{
				Host:    targetHost,
				List:    rblList,
				Results: []godnsbl.Result{{Listed: true, Address: targetHost, Text: "Some Error", Rbl: rblList}},
			}
		}

		Convey("Panic on invalid number of workers", func() {
			So(func() {
				newDnsChecker(Options{
					Lookup:           lookup,
					NumberOfWorkers:  0, // cannot have less than 1 worker!
					CheckedAddress:   net.ParseIP("11.22.33.44"),
					RBLProvidersURLs: []string{"rbl1", "rbl2", "rbl3", "rbl4", "rbl5"},
				})
			}, ShouldPanic)
		})

		Convey("Not blocked in any lists", func() {
			checker := newDnsChecker(Options{
				Lookup:           lookup,
				NumberOfWorkers:  2,
				CheckedAddress:   net.ParseIP("11.22.33.44"),
				RBLProvidersURLs: []string{"rbl1", "rbl2", "rbl3", "rbl4", "rbl5"},
			})

			defer checker.Close()

			checker.StartListening()

			baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)

			checker.NotifyNewScan(baseTime)

			time.Sleep(700 * time.Millisecond)

			select {
			case <-checker.checkerResultsChan:
				So(false, ShouldBeTrue)
			default:
			}
		})

		Convey("Blocked in some RBLs", func() {
			checker := newDnsChecker(Options{
				Lookup:           lookup,
				NumberOfWorkers:  2,
				CheckedAddress:   net.ParseIP("11.22.33.44"),
				RBLProvidersURLs: []string{"rbl1-blocked", "rbl2", "rbl3-blocked", "rbl4-blocked", "rbl5"},
			})

			defer checker.Close()

			checker.StartListening()

			baseTime := testutil.MustParseTime(`2000-01-01 00:00:00 +0000`)

			checker.NotifyNewScan(baseTime)

			time.Sleep(700 * time.Millisecond)

			select {
			case r := <-checker.checkerResultsChan:
				So(r.RBLs, ShouldResemble, []ContentElement{
					{RBL: "rbl1-blocked", Text: "Some Error"},
					{RBL: "rbl3-blocked", Text: "Some Error"},
					{RBL: "rbl4-blocked", Text: "Some Error"},
				})

				So(r.Interval.From, ShouldResemble, baseTime)
				So(r.Interval.To.After(r.Interval.From), ShouldBeTrue)
			default:
				So(false, ShouldBeTrue)
			}
		})
	})
}
