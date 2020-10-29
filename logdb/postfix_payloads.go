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
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"log"
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
	// Apply changes on database with name x
	database string
	// Name of migration file
	filename string
	// Delete create the database tables, indexes, etc.
	up func(tx *sql.Tx) error
	// Register Delete the database tables, indexes, etc.
	down func(tx *sql.Tx) error

	// Counts how many records are there in the respective table.
	counter func(dbconn.RoConn) int

	// Finds the timestamp for the most recent inserted in the table.
	lastTimeReader func(dbconn.RoConn) (int64, error)
}

var (
	payloadHandlers []payloadHandler
)

func registerPayloadHandler(handler payloadHandler) {
	if handler.down == nil {
		log.Panicln("Down func is nil")
	}

	if handler.up == nil {
		log.Panicln("Down func is nil")
	}

	if handler.filename == "" {
		log.Panicln("filename is empty")
	}

	if handler.database == "" {
		log.Panicln("database name is empty")
	}

	migrator.AddMigration(handler.database, handler.filename, handler.up, handler.down)

	payloadHandlers = append(payloadHandlers, handler)
}
