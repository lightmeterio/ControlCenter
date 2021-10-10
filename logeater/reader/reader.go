// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package reader

import (
	"bufio"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
	"time"
)

func min(a, b uint) uint {
	if a < b {
		return a
	}

	return b
}

// TODO: unit test this function and try to find edge cases, as there are possibly many!
func ReadFromReader(reader io.Reader, pub postfix.Publisher, builder transform.Builder, importAnnouncer announcer.ImportAnnouncer, clock timeutil.Clock, timeout time.Duration) error {
	t, err := builder()
	if err != nil {
		return errorutil.Wrap(err)
	}

	// when importing logs from the past (duh!) we expect that the importing progress
	// ends when we reach the moment where the import was triggered,
	// otherwise we will never know when it ends.
	expectedImportEndTime := clock.Now()

	// a totally arbitrary number. It could be anything <= 100
	const numberOfSteps = 100

	var (
		completed           [numberOfSteps]bool
		firstLine           = true
		initialTime         time.Time
		endAlreadyAnnounced = false
		currentRecord       postfix.Record
		hasher              = postfix.NewHasher()
	)

	progress := func(t time.Time) uint {
		if expectedImportEndTime.Unix() == initialTime.Unix() {
			return 100
		}

		v := ((t.Unix() - initialTime.Unix()) * numberOfSteps) / (expectedImportEndTime.Unix() - initialTime.Unix())

		return uint(v)
	}

	announceEnd := func(t time.Time) {
		importAnnouncer.AnnounceProgress(announcer.Progress{
			Finished: true,
			Time:     t,
			Progress: 100,
		})

		endAlreadyAnnounced = true
	}

	setupAnnouncerIfNeeded := func(r postfix.Record) {
		if !firstLine {
			return
		}

		firstLine = false
		initialTime = r.Time

		importAnnouncer.AnnounceStart(r.Time)
	}

	announceProgressIfPossible := func(t time.Time) {
		p := progress(t)

		if completed[min(p, numberOfSteps-1)] {
			return
		}

		completed[min(p, numberOfSteps-1)] = true

		importAnnouncer.AnnounceProgress(announcer.Progress{
			Finished: false,
			Time:     t,
			Progress: int64(p),
		})
	}

	handleNewLogLine := func(line []byte) {
		var err error

		currentRecord, err = t.Transform(line)
		if err != nil {
			log.Err(err).Msgf("Error reading from reader: %v", err)
			return
		}

		currentRecord.Sum = postfix.ComputeChecksum(hasher, currentRecord)

		setupAnnouncerIfNeeded(currentRecord)

		pub.Publish(currentRecord)

		if endAlreadyAnnounced {
			return
		}

		isPastImportEndTime := currentRecord.Time.After(expectedImportEndTime)

		if !isPastImportEndTime {
			announceProgressIfPossible(currentRecord.Time)
			return
		}

		announceEnd(currentRecord.Time)
	}

	buildEndAnnounceTime := func() time.Time {
		if !currentRecord.Time.IsZero() {
			return currentRecord.Time
		}

		return expectedImportEndTime
	}

	linesChan := make(chan []byte)
	continueScanning := make(chan struct{})
	doneScanning := make(chan struct{})

	scanner := bufio.NewScanner(reader)

	go func() {
		for scanner.Scan() {
			linesChan <- scanner.Bytes()

			<-continueScanning
		}

		doneScanning <- struct{}{}
	}()

	timer := time.NewTimer(timeout)

loop:
	for {
		// especial case where, when the reader does not read anything after some time,
		// we assume all the old logs to be received have already been received,
		// and are ready now to receive new logs.
		select {
		case <-timer.C:
			if endAlreadyAnnounced {
				break
			}

			if firstLine {
				break
			}

			announceEnd(buildEndAnnounceTime())
		case line := <-linesChan:
			handleNewLogLine(line)
			continueScanning <- struct{}{}
		case <-doneScanning:
			break loop
		}
	}

	// special case where the reader is empty and no log is sent
	if firstLine {
		importAnnouncer.AnnounceStart(buildEndAnnounceTime())
	}

	if endAlreadyAnnounced {
		return nil
	}

	announceEnd(buildEndAnnounceTime())

	return nil
}
