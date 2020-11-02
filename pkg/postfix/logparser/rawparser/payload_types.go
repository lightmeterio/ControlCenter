package rawparser

type PayloadType int

const (
	PayloadTypeUnsupported PayloadType = iota
	PayloadTypeQmgrReturnedToSender
	PayloadTypeSmtpMessageStatus
)
