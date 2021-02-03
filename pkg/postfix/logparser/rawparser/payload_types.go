// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package rawparser

type PayloadType int

const (
	PayloadTypeUnsupported PayloadType = iota
	PayloadTypeQmgrReturnedToSender
	PayloadTypeQmgrMailQueued
	PayloadTypeQmgrRemoved
	PayloadTypeSmtpMessageStatus
	PayloadTypeSmtpdConnect
	PayloadTypeSmtpdDisconnect
	PayloadTypeSmtpdMailAccepted
	PayloadTypeCleanupMessageAccepted
	PayloadTypeBounceCreated
	PayloadTypePickup

	// types for SmtpMessageStatus extra message
	PayloadTypeSmtpMessageStatusSentQueued
)
