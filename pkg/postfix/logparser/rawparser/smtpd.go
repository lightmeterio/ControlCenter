// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:generate ragel -Z -G2 smtpd.rl -o smtpd.gen.go

package rawparser

func init() {
	// from the standard postfix setup
	registerHandler("postfix", "submission/smtpd", parseSmtpdPayload) // for STARTTLS submission connection (port 587)
	registerHandler("postfix", "smtps/smtpd", parseSmtpdPayload)      // for TLS submission connection (port 465)
	registerHandler("postfix", "smtpd", parseSmtpdPayload)            // for MTA connection (port 25)

	// detected in some zimbra setups
	registerHandler("postfix", "amavisd/smtpd", parseSmtpdPayload)
	registerHandler("postfix", "dkimmilter/smtpd", parseSmtpdPayload)
	registerHandler("postfix", "smtps/smtpd", parseSmtpdPayload)
}

type SmtpdConnect struct {
	Host string
	IP   string
}

type SmtpdDisconnectStat struct {
	Success int
	Total   int
}

type SmtpdDisconnect struct {
	Host  string
	IP    string
	Stats map[string]SmtpdDisconnectStat
}

type SmtpdMailAccepted struct {
	Host  string
	IP    string
	Queue string
}

func parseSmtpdPayload(payloadLine string) (RawPayload, error) {
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

	if s, parsed := parseSmtpdReject(payloadLine); parsed {
		return RawPayload{
			PayloadType: PayloadTypeSmtpdReject,
			SmtpdReject: s,
		}, nil
	}

	return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
}

type SmtpdReject struct {
	Queue        string
	ExtraMessage string
}
