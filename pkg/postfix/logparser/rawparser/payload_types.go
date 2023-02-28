// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

type PayloadType int

const (
	PayloadTypeUnsupported PayloadType = iota
	PayloadTypeQmgrMessageExpired
	PayloadTypeQmgrMailQueued
	PayloadTypeQmgrRemoved
	PayloadTypeSmtpMessageStatus
	PayloadTypeSmtpdConnect
	PayloadTypeSmtpdDisconnect
	PayloadTypeSmtpdMailAccepted
	PayloadTypeSmtpdReject
	PayloadTypeCleanupMessageAccepted
	PayloadTypeBounceCreated
	PayloadTypePickup
	PayloadTypeCleanupMilterReject
	PayloadTypeVersion
	PayloadTypeDovecotAuthFailedWithReason
	PayloadTypeLightmeterDumpedHeader
	PayloadTypeLightmeterRelayedBounce

	// types for SmtpMessageStatus extra message
	PayloadTypeSmtpMessageStatusSentQueued
	PayloadSmtpSentStatusExtraMessageNewUUID
)
