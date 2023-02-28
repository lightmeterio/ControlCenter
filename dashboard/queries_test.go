// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dashboard

import (
	"context"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"log"
	"testing"
	"time"
)

func mustEncode(i interface{}) string {
	b, err := json.Marshal(i)
	So(err, ShouldBeNil)
	return string(b)
}

type fakeQuery struct {
	rows [][]interface{}
}

func (q *fakeQuery) QueryContext(ctx context.Context, args ...interface{}) (QueryableRows, error) {
	return &fakeQueryRows{
		q:    q,
		args: args,
	}, nil
}

type fakeQueryRows struct {
	index int
	q     *fakeQuery
	args  []interface{}
}

func (q *fakeQueryRows) Scan(args ...interface{}) error {
	for i, arg := range q.q.rows[q.index] {
		switch a := arg.(type) {
		case string:
			*(args[i].(*string)) = a
		case int64:
			*(args[i].(*int64)) = a
		case []byte:
			*(args[i].(*[]byte)) = a
		default:
			log.Panicf("Converting value '%#v' is not supported", arg)
		}
	}

	return nil
}

func (q *fakeQueryRows) ForEach(f func(QueryableScanner) error) error {
	for range q.q.rows {
		if err := f(q); err != nil {
			return err
		}

		q.index++
	}

	return nil
}

func (q *fakeQueryRows) Close() error {
	return nil
}

func TestQueries(t *testing.T) {
	Convey("Test Queries", t, func() {
		q := &fakeQuery{
			rows: [][]interface{}{
				[]interface{}{
					"user1@example.com", timeutil.MustParseTime(`2020-01-01 00:00:00 +0000`).Unix(), timeutil.MustParseTime(`2020-01-06 00:00:00 +0000`).Unix(),
					mustEncode([][2]int64{
						{timeutil.MustParseTime(`2020-01-01 00:00:00 +0000`).Unix(), 10},
						{timeutil.MustParseTime(`2020-01-04 00:00:00 +0000`).Unix(), 10},
						{timeutil.MustParseTime(`2020-01-06 00:00:00 +0000`).Unix(), 50},
					}),
				},
				[]interface{}{
					"user2@anotherexample.com", timeutil.MustParseTime(`2020-01-02 00:00:00 +0000`).Unix(), timeutil.MustParseTime(`2020-01-09 00:00:00 +0000`).Unix(),
					mustEncode([][2]int64{
						{timeutil.MustParseTime(`2020-01-02 00:00:00 +0000`).Unix(), 2},
						{timeutil.MustParseTime(`2020-01-09 00:00:00 +0000`).Unix(), 20},
					}),
				},
			},
		}

		const oneDayInHours = 24 * int(time.Hour/time.Second)

		r, err := queryMailTrafficPerMailboxWithQueryable(context.Background(), q, oneDayInHours)
		So(err, ShouldBeNil)

		So(r, ShouldResemble, MailTrafficPerSenderOverTimeResult{
			Times: []int64{
				timeutil.MustParseTime(`2020-01-01 00:00:00 +0000`).Unix(),
				timeutil.MustParseTime(`2020-01-02 00:00:00 +0000`).Unix(),
				timeutil.MustParseTime(`2020-01-03 00:00:00 +0000`).Unix(),
				timeutil.MustParseTime(`2020-01-04 00:00:00 +0000`).Unix(),
				timeutil.MustParseTime(`2020-01-05 00:00:00 +0000`).Unix(),
				timeutil.MustParseTime(`2020-01-06 00:00:00 +0000`).Unix(),
				timeutil.MustParseTime(`2020-01-07 00:00:00 +0000`).Unix(),
				timeutil.MustParseTime(`2020-01-08 00:00:00 +0000`).Unix(),
				timeutil.MustParseTime(`2020-01-09 00:00:00 +0000`).Unix(),
			},
			Values: map[string][]int64{
				"user1@example.com":        []int64{10, 0, 0, 10, 0, 50, 0, 0, 0},
				"user2@anotherexample.com": []int64{0, 2, 0, 0, 0, 0, 0, 0, 20},
			},
		})
	})
}
