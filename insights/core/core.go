package core

import (
	"database/sql"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/i18n/translator"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
}

type Detector interface {
	Step(Clock, *sql.Tx) error
	Close() error
}

type Core struct {
	Detectors []Detector
	closers   closeutil.Closers
}

func New(detectors []Detector) (*Core, error) {
	Detectors := []Detector{}
	closers := closeutil.Closers{}

	for _, d := range detectors {
		Detectors = append(Detectors, d)
		closers = append(closers, d)
	}

	return &Core{
		Detectors: Detectors,
		closers:   closers,
	}, nil
}

func (c *Core) Close() error {
	if err := c.closers.Close(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type Content interface {
	fmt.Stringer
	translator.TranslatableStringer
}
