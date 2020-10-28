package main

import "gitlab.com/lightmeter/controlcenter/tools/gotestdata/translator"

func I18n(s string) string {
	return ""
}

type DataA struct{}

func (d *DataA) String(s string) string {
	return I18n("DataA")
}

type DataB struct{}

func (d *DataB) String(s string) string {
	b := I18n("DataB")

	return b
}

func main() {
	I18n("hello world")

	i := I18n
	i("Mister Donutloop")

	translator.I18n("DL")
}
