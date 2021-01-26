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

	// types for SmtpMessageStatus extra message
	PayloadTypeSmtpMessageStatusSentQueued
)
