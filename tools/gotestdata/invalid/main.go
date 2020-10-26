package main

func I18n(s string) string {
	return ""
}

type DataC struct{}

func (d *DataC) String(s string) string {
	a := "DataC"
	b := I18n(a)

	return b
}


