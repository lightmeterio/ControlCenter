// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package collector

import (
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"

	"github.com/rs/zerolog/log"

	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var ErrUnregisteredUser = errors.New(`Can't send reports, user is not registered yet`)

type Dispatcher interface {
	Dispatch(Report) error
}

type ReportPayload interface{}

type ReportEntry struct {
	Time    time.Time     `json:"time"`
	ID      string        `json:"id"`
	Payload ReportPayload `json:"payload"`
}

type Report struct {
	Interval timeutil.TimeInterval `json:"interval"`
	Content  []ReportEntry         `json:"content"`
}

func lastReportTime(tx *sql.Tx) (time.Time, error) {
	var ts int64

	err := tx.QueryRow(`select time from dispatch_times order by id desc limit 1`).Scan(&ts)

	// first execution. Initial time undefined
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}

	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	return time.Unix(ts, 0).In(time.UTC), nil
}

func TryToDispatchReports(tx *sql.Tx, clock timeutil.Clock, dispatcher Dispatcher) (err error) {
	// creates a report and mark all the queued reports as dispatched
	// TODO: maybe do the dispatching in a different thread,
	// in order not to block the transaction?
	//nolint:sqlclosecheck
	r, err := tx.Query(`select time, identifier, value from queued_reports where dispatched_time = 0 order by id asc`)
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer errorutil.UpdateErrorFromCloser(r, &err)

	initialTime, err := lastReportTime(tx)
	if err != nil {
		return errorutil.Wrap(err)
	}

	now := clock.Now()

	report := Report{Interval: timeutil.TimeInterval{From: initialTime, To: now}}

	for r.Next() {
		var (
			ts   int64
			id   string
			blob string
		)

		if err := r.Scan(&ts, &id, &blob); err != nil {
			return errorutil.Wrap(err)
		}

		time := time.Unix(ts, 0).In(time.UTC)

		var value interface{}

		if err := json.Unmarshal([]byte(blob), &value); err != nil {
			return errorutil.Wrap(err)
		}

		report.Content = append(report.Content, ReportEntry{Time: time, ID: id, Payload: value})
	}

	if err := r.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	if len(report.Content) == 0 {
		log.Debug().Msgf("Nothing to be reported!")
		return nil
	}

	log.Info().Msgf("Dispatching reports at %v with last dispatch time %v", now, initialTime)

	if err := dispatcher.Dispatch(report); err != nil {
		if errors.Is(err, ErrUnregisteredUser) {
			log.Info().Msgf("Can't send reports yet, user not registered")
			return nil
		}

		return errorutil.Wrap(err)
	}

	if err := storeDispatchTime(tx, now); err != nil {
		return errorutil.Wrap(err)
	}

	if _, err := tx.Exec(`update queued_reports set dispatched_time = ? where dispatched_time = 0`, now.Unix()); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func storeDispatchTime(tx *sql.Tx, time time.Time) error {
	if _, err := tx.Exec(`insert into dispatch_times(time) values(?)`, time.Unix()); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func Step(tx *sql.Tx, clock timeutil.Clock, reporters Reporters, dispatcher Dispatcher, dispatchingInterval time.Duration) error {
	if err := reporters.Step(tx, clock); err != nil {
		return errorutil.Wrap(err)
	}

	lastDispatchTime, err := lastReportTime(tx)
	if err != nil {
		return errorutil.Wrap(err)
	}

	isFirstReportEver := lastDispatchTime.IsZero()
	now := clock.Now()
	timeSinceLastReport := now.Sub(lastDispatchTime)
	timeSinceLastReportElapsed := timeSinceLastReport >= dispatchingInterval

	if !(isFirstReportEver || timeSinceLastReportElapsed) {
		return nil
	}

	if err := TryToDispatchReports(tx, clock, dispatcher); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
