/**
 * This file describes how to add support for storing a new log type in the database
 * You'll need to add a new entry on the switch-case on FindInserterForPayload()
 * and register the other functions for the new log type with the function registerPayloadHandler()
 * in the respective init(). For an example, please check the file postfix_payload_smtp.go
 */

package logdb

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	parser "gitlab.com/lightmeter/postfix-log-parser"
)

func findInserterForPayload(payload parser.Payload) func(*sql.Tx, data.Record) error {
	// NOTE: I wish go had a way to use type information as a map key
	// so this switch could be simplified to registering handlers for payloads at runtime
	// instead of this ugly switch case!
	switch payload.(type) {
	case parser.SmtpSentStatus:
		return inserterForSmtpSentStatus
	}

	return nil
}

type payloadHandler struct {
	// Create the database tables, indexes, etc.
	creator func(dbconn.RwConn) error

	// Counts how many records are there in the respective table.
	counter func(dbconn.RoConn) int

	// Finds the timestamp for the most recent inserted in the table.
	lastTimeReader func(dbconn.RoConn) (int64, error)
}

var (
	payloadHandlers []payloadHandler
)

func registerPayloadHandler(handler payloadHandler) {
	payloadHandlers = append(payloadHandlers, handler)
}
