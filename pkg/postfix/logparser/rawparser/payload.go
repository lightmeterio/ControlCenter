package rawparser

// NOTE: Go does not have unions, and using interfaces implies on virtual calls
// (which are being done in the higher level parsing interface, anyways),
// so we add all the possible payloads inlined in the struct, with a field describing which
// payload the whole record refers to.
// This is ok as all payloads here store basically byte slices only, which are trivially constructible and copyable
// so, although this struct will grow as newer payloads are supported,
// copying will perform better than using virtual calls
type RawPayload struct {
	PayloadType          PayloadType
	RawSmtpSentStatus    RawSmtpSentStatus
	QmgrReturnedToSender QmgrReturnedToSender
}
