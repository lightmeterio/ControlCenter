// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package announcer

import (
	"sync"
	"time"
)

type Progress struct {
	Finished bool
	Time     time.Time
	Progress int64
}

type ImportAnnouncer interface {
	AnnounceStart(time.Time)
	AnnounceProgress(Progress)
}

type Notifier struct {
	currentProgress int
	stepValue       int
	announcer       ImportAnnouncer
}

func NewNotifier(announcer ImportAnnouncer, steps int) Notifier {
	return Notifier{
		currentProgress: 0,
		stepValue:       100 / steps,
		announcer:       announcer,
	}
}

func (p *Notifier) Start(t time.Time) {
	p.announcer.AnnounceStart(t)
}

func (p *Notifier) End(t time.Time) {
	p.announcer.AnnounceProgress(Progress{
		Finished: true,
		Time:     t,
		Progress: 100,
	})
}

func clamp(v int) int {
	if v < 100 {
		return v
	}

	return 100
}

func (p *Notifier) Step(t time.Time) {
	p.currentProgress += p.stepValue

	p.announcer.AnnounceProgress(Progress{
		Finished: false,
		Time:     t,
		Progress: int64(clamp(p.currentProgress)),
	})
}

// Skipper allows progress announcers to be skipt
func Skipper(announcer ImportAnnouncer) ImportAnnouncer {
	return &skipper{announcer: announcer, ended: false}
}

type skipper struct {
	announcer ImportAnnouncer
	ended     bool
}

func (s *skipper) AnnounceStart(time.Time) {
	s.announcer.AnnounceStart(time.Time{})
}

func (s *skipper) AnnounceProgress(Progress) {
	if s.ended {
		return
	}

	s.ended = true
	s.announcer.AnnounceProgress(Progress{Finished: true, Time: time.Time{}, Progress: 100})
}

// Skip an import, notifying it's finished as soon as it starts
func Skip(announcer ImportAnnouncer) {
	s := Skipper(announcer)
	s.AnnounceStart(time.Time{})
	s.AnnounceProgress(Progress{})
}

type DummyImportAnnouncer struct {
	sync.Mutex
	Start    time.Time
	progress []Progress
}

func (a *DummyImportAnnouncer) AnnounceStart(t time.Time) {
	a.Lock()

	defer a.Unlock()

	a.Start = t
}

func (a *DummyImportAnnouncer) AnnounceProgress(p Progress) {
	a.Lock()

	defer a.Unlock()

	a.progress = append(a.progress, p)
}

func (a *DummyImportAnnouncer) Progress() []Progress {
	a.Lock()

	defer a.Unlock()

	c := make([]Progress, len(a.progress))

	if i := copy(c, a.progress); i != len(a.progress) {
		panic(i)
	}

	return c
}

// does nothing
type EmptyImportAnnouncer struct{}

func (*EmptyImportAnnouncer) AnnounceStart(time.Time) {
	// it's empty...
}

func (*EmptyImportAnnouncer) AnnounceProgress(Progress) {
	// it's empty...
}
