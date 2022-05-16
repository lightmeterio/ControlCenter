// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"database/sql"
	"errors"
	"time"

	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
)

type NodeType int

const (
	SingleNodeType NodeType = iota
	AuthenticatedMultiNode
)

type NodeTypeHandler interface {
	CreateQueue(time.Time, int64, string, postfix.RecordLocation, dbconn.TxPreparedStmts) (int64, error)
	FindQueue(string, postfix.Record, dbconn.TxPreparedStmts) (int64, error)
	HandleMailSentAction(*sql.Tx, postfix.Record, parser.SmtpSentStatus, dbconn.TxPreparedStmts) error
}

var ErrInvalidNodeType = errors.New(`Invalid Node Type`)

func BuildNodeTypeHandler(typ string) (NodeTypeHandler, error) {
	switch typ {
	case "single":
		return &SingleNodeTypeHandler{}, nil
	case "multi":
		return &MultiNodeTypeHandler{}, nil
	}

	return nil, ErrInvalidNodeType
}
