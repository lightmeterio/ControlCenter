// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package intel

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/deliverydb"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/intel/mailactivity"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"os"
	"time"
)

type dispatcher struct {
}

func (d *dispatcher) Dispatch(r collector.Report) error {
	// TODO: this guy here is responsible for connecting to send the report to our server...
	// TODO: implement it instead of saving to a file!!!
	log.Info().Msgf("Sending a new Network intelligence report in the interval %v and with %v rows", r.Interval, len(r.Content))

	json, err := json.Marshal(r)
	if err != nil {
		return errorutil.Wrap(err)
	}

	filename := fmt.Sprintf("/tmp/lightmeter-report-%v.json", time.Now().Unix())

	if err := os.WriteFile(filename, json, os.ModeAppend); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func New(workspaceDir string, db *deliverydb.DB) (*collector.Collector, error) {
	reporters := collector.Reporters{
		mailactivity.NewReporter(db.ConnPool()),
	}

	options := collector.Options{
		CycleInterval:  time.Minute * 1,
		ReportInterval: time.Minute * 30,
	}

	dispatcher := &dispatcher{}

	c, err := collector.New(workspaceDir, options, reporters, dispatcher)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return c, nil
}
