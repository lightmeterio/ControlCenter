package logsource

import (
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type Reader struct {
	source Source
	pub    data.Publisher
}

func NewReader(source Source, pub data.Publisher) Reader {
	return Reader{source: source, pub: pub}
}

func (r *Reader) Run() error {
	if err := r.source.PublishLogs(r.pub); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
