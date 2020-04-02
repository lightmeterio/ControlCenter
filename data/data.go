package data

import (
	parser "gitlab.com/lightmeter/postfix-log-parser"
)

type Record struct {
	Header  parser.Header
	Payload parser.Payload
}

type Publisher interface {
	Publish(Record)
	Close()
}

type ChannelBasedPublisher struct {
	channel chan<- Record
}

func (pub *ChannelBasedPublisher) Publish(status Record) {
	pub.channel <- status
}

func (pub *ChannelBasedPublisher) Close() {
	close(pub.channel)
}
