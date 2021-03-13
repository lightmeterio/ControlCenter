// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package announcer

import (
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
	currentProgress int64
	stepValue       int64
	announcer       ImportAnnouncer
}

func NewNotifier(announcer ImportAnnouncer, step int64) Notifier {
	return Notifier{
		currentProgress: 0,
		stepValue:       step,
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

func (p *Notifier) Step(t time.Time) {
	p.currentProgress += p.stepValue

	p.announcer.AnnounceProgress(Progress{
		Finished: false,
		Time:     t,
		Progress: p.currentProgress,
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
