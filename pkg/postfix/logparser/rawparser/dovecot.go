// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:generate ragel -Z -G2 dovecot.rl -o dovecot.gen.go

package rawparser

func init() {
	registerHandler("dovecot", "", parseDovecotPayload)
}

type DovecotAuthFailedWithReason struct {
	DB       string
	Username string
	IP       string

	// Different reasons
	DovecotAuthFailedReasonPasswordMismatch  string
	DovecotAuthFailedReasonUnknownUser       string
	DovecotAuthFailedReasonAuthPolicyRefusal string

	ReasonExplanation string
}

func parseDovecotPayload(payloadLine string) (RawPayload, error) {
	if p, parsed := parseDovecotAuthFailedWithReason(payloadLine); parsed {
		return RawPayload{
			PayloadType:                 PayloadTypeDovecotAuthFailedWithReason,
			DovecotAuthFailedWithReason: p,
		}, nil
	}

	return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
}
