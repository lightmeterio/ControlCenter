package localrbl

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/data"
	"io"
	"net"
	"time"
)

type Options struct {
	NumberOfWorkers  int
	RBLProvidersURLs []string
	Lookup           DNSLookupFunction
}

type ContentElement struct {
	RBL  string `json:"rbl"`
	Text string `json:"text"`
}

type Results struct {
	Err      error
	Interval data.TimeInterval
	RBLs     []ContentElement
}

type Checker interface {
	io.Closer
	StartListening()
	NotifyNewScan(time.Time)
	Step(time.Time, func(Results) error, func() error) error
	CheckedIP(context.Context) net.IP
}
