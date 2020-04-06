package data

import (
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"time"
)

// The parsed value as read from the logs
// TODO: add the timestamp of when the log was read
// ir order to improve detecting inconsistences on reading
// from multiple sources
type Record struct {
	Header  parser.Header
	Payload parser.Payload
}

// A record with the actual timestamp it'll be used to be inserted in the database
type TimedRecord struct {
	Time   time.Time
	Record Record
}

type Publisher interface {
	Publish(Record)
	Close()
}

type Config struct {
	Location    *time.Location
	DefaultYear int
}
