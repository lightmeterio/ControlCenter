// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package topdomains

import (
	"context"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/intel/collector"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

type report struct {
	SenderDomains    []string `json:"senders"`
	RecipientDomains []string `json:"recipients"`
}

type Reporter struct {
	pool *dbconn.RoPool
}

func NewReporter(pool *dbconn.RoPool) *Reporter {
	return &Reporter{pool: pool}
}

const executionInterval = 1 * time.Hour

func (r *Reporter) ExecutionInterval() time.Duration {
	return executionInterval
}

func (r *Reporter) Close() error {
	return nil
}

const reporterId = "top_domains"
const alreadyExecutedFlag = "top_domains_already_executed"

func buildTimeIntervalCondition(tx *sql.Tx, clock timeutil.Clock) (query string, args []interface{}, err error) {
	defer func() {
		if err != nil {
			return
		}

		if sErr := metadata.Store(context.Background(), tx, []metadata.Item{{Key: alreadyExecutedFlag, Value: true}}); sErr != nil {
			err = sErr
		}
	}()

	var alreadyExecuted bool

	err = metadata.Retrieve(context.Background(), tx, alreadyExecutedFlag, &alreadyExecuted)
	if err != nil && !errors.Is(err, metadata.ErrNoSuchKey) {
		return "", nil, errorutil.Wrap(err)
	}

	// there was a prior execution. Use time in the interval
	if errors.Is(err, metadata.ErrNoSuchKey) {
		// First execution. Use all data available (it can be quite slow in existing deployments with large of data in the database)
		return `and deliveries.delivery_ts <= ?`, []interface{}{clock.Now().Unix()}, nil
	}

	now := clock.Now()

	return `and deliveries.delivery_ts >= ? and deliveries.delivery_ts <= ?`, []interface{}{now.Add(-executionInterval).Unix(), now.Unix()}, nil
}

func fillDomains(conn *dbconn.RoPooledConn, query string, args []interface{}, direction tracking.MessageDirection) (domains []string, err error) {
	domains = []string{}

	argsWithDirection := append([]interface{}{int64(direction)}, args...)

	//nolint:sqlclosecheck
	r, err := conn.Query(query, argsWithDirection...)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer errorutil.UpdateErrorFromCloser(r, &err)

	for r.Next() {
		var (
			// count is ignored for now
			count  int64
			domain string
		)

		if err := r.Scan(&count, &domain); err != nil {
			return nil, errorutil.Wrap(err)
		}

		domains = append(domains, domain)
	}

	if err := r.Err(); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return domains, nil
}

func (r *Reporter) Step(tx *sql.Tx, clock timeutil.Clock) error {
	intervalCondition, args, err := buildTimeIntervalCondition(tx, clock)
	if err != nil {
		return errorutil.Wrap(err)
	}

	report := report{}

	conn, release := r.pool.Acquire()

	defer release()

	topSenderQuery := `
	select
		count(*) as c, remote_domains.domain
	from
		deliveries join remote_domains on sender_domain_part_id = remote_domains.id
	where
		deliveries.direction = ? ` + intervalCondition + ` and length(remote_domains.domain) > 0
	group by
		remote_domains.domain
	order by
		c desc
	`

	topRecipientQuery := `
	select
		count(*) as c, remote_domains.domain
	from
		deliveries join remote_domains on recipient_domain_part_id = remote_domains.id
	where
		deliveries.direction = ? ` + intervalCondition + ` and length(remote_domains.domain) > 0
	group by
		remote_domains.domain
	order by
		c desc
	`

	report.SenderDomains, err = fillDomains(conn, topSenderQuery, args, tracking.MessageDirectionOutbound)
	if err != nil {
		return errorutil.Wrap(err)
	}

	report.RecipientDomains, err = fillDomains(conn, topRecipientQuery, args, tracking.MessageDirectionIncoming)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if len(report.SenderDomains)+len(report.RecipientDomains) == 0 {
		return nil
	}

	if err := collector.Collect(tx, clock, r.ID(), &report); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (r *Reporter) ID() string {
	return reporterId
}
