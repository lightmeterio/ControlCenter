package core

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/util"
	"time"
)

type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
}

type Setterup interface {
	Setup(*sql.Tx) error
}

type StepperWithSetup interface {
	Stepper
	Setterup
}

type Detector interface {
	Setterup
	Steppers() []Stepper
}

type Stepper interface {
	Step(Clock, *sql.Tx) error
	Close() error
}

type Core struct {
	Steppers []Stepper
}

func New(detectors []Detector) (*Core, error) {
	Steppers := []Stepper{}

	for _, d := range detectors {
		Steppers = append(Steppers, d.Steppers()...)
	}

	return &Core{
		Steppers: Steppers,
	}, nil
}

func (c *Core) Close() error {
	// close steppers in reverse order, as the detectors always come before any object it owns
	for i := len(c.Steppers) - 1; i >= 0; i-- {
		if err := c.Steppers[i].Close(); err != nil {
			return util.WrapError(err)
		}
	}

	return nil
}

type Content interface{}
