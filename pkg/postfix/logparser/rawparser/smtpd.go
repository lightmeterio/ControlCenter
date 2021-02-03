// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

//go:generate ragel -Z -G2 smtpd.rl -o smtpd.gen.go

package rawparser

func init() {
	registerHandler("postfix", "submission/smtpd", parseSmtpdPayload) // for remote connection
	registerHandler("postfix", "smtpd", parseSmtpdPayload)            // for local connection
	registerHandler("postfix", "amavisd/smtpd", parseSmtpdPayload)    // for some weird setups (zimbra, for instance)
}

type SmtpdConnect struct {
	Host []byte
	IP   []byte
}

type SmtpdDisconnect struct {
	Host []byte
	IP   []byte
}

type SmtpdMailAccepted struct {
	Host  []byte
	IP    []byte
	Queue []byte
}

func parseSmtpdPayload(header RawHeader, payloadLine []byte) (RawPayload, error) {
	if s, parsed := parseSmtpdConnect(payloadLine); parsed {
		return RawPayload{
			PayloadType:  PayloadTypeSmtpdConnect,
			SmtpdConnect: s,
		}, nil
	}

	if s, parsed := parseSmtpdDisconnect(payloadLine); parsed {
		return RawPayload{
			PayloadType:     PayloadTypeSmtpdDisconnect,
			SmtpdDisconnect: s,
		}, nil
	}

	if s, parsed := parseSmtpdMailAccepted(payloadLine); parsed {
		return RawPayload{
			PayloadType:       PayloadTypeSmtpdMailAccepted,
			SmtpdMailAccepted: s,
		}, nil
	}

	return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
}
