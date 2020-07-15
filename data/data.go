package data

import (
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"time"
)

type Record struct {
	Time    time.Time
	Header  parser.Header
	Payload parser.Payload
}

type Publisher interface {
	Publish(Record)
	Close()
}

type Config struct {
	Location *time.Location
}
