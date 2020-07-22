package domainmapping

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestMapping(t *testing.T) {
	Convey("Test Mapping", t, func() {
		Convey("Empty Mapping", func() {
			l, err := Mapping(RawList{})
			So(err, ShouldBeNil)
			So(l.Resolve(""), ShouldEqual, "")
			So(l.Resolve("example.com"), ShouldEqual, "example.com")
		})

		Convey("Some grouping", func() {
			l, err := Mapping(RawList{
				"example":  []string{"example.com", "beispiel.de"},
				"provider": []string{"provider.com", "provider.de"},
			})

			So(err, ShouldBeNil)
			So(l.Resolve("example.com"), ShouldEqual, "example")
			So(l.Resolve("beispiel.de"), ShouldEqual, "example")
			So(l.Resolve("exemplo.com.br"), ShouldEqual, "exemplo.com.br")
		})

		Convey("Search is case insensitive", func() {
			l, err := Mapping(RawList{
				"example": []string{"example.com", "beispiel.de"},
			})

			So(err, ShouldBeNil)
			So(l.Resolve("EXAMPLE.COM"), ShouldEqual, "example")
			So(l.Resolve("BEISPIEL.DE"), ShouldEqual, "example")
			So(l.Resolve("SomethingElse.uk"), ShouldEqual, "somethingelse.uk")
		})

		Convey("Duplicate mapping", func() {
			_, err := Mapping(RawList{
				"example":  []string{"example.com", "beispiel.de"},
				"provider": []string{"provider.com", "example.com"},
			})

			So(err, ShouldNotBeNil)
		})

	})
}
