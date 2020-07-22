package domainmapping

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestMapping(t *testing.T) {
	Convey("Test Mapping", t, func() {
		Convey("Empty Mapping", func() {
			l := Mapping(RawList{})
			So(l.Resolve(""), ShouldEqual, "")
			So(l.Resolve("example.com"), ShouldEqual, "example.com")
		})

		Convey("Some grouping", func() {
			l := Mapping(RawList{
				"example":  []string{"example.com", "beispiel.de"},
				"provider": []string{"provider.com", "beispiel.de"},
			})
			So(l.Resolve("example.com"), ShouldEqual, "example")
			So(l.Resolve("beispiel.de"), ShouldEqual, "example")
			So(l.Resolve("exemplo.com.br"), ShouldEqual, "exemplo.com.br")
		})
	})
}
