// +build codeanalysis

/**
 * Add empty versions of all ragel generated functions to this file,
 * so that golangci-lint will not fail as such functions don't exist before the code generation.
 * As we don't include the generate code in the repository, golangci-lint will ignore them.
 */

package rawparser

// actual implementation on header.rl
func parseHeaderPostfixPart(*RawHeader, []byte) (int, bool) { panic("NOOP") }

// actual implementation on smtp.rl
func parseSmtpSentStatus([]byte) (RawSmtpSentStatus, bool) { panic("NOOP") }

// actual implementation on qmgr.rl
func parseQmgrReturnedToSender([]byte) (QmgrReturnedToSender, bool) { panic("NOOP") }
