// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package rawparser

func init() {
	registerHandler("postfix", "smtp", parseSmtpPayload)
	registerHandler("postfix", "lmtp", parseSmtpPayload)
	registerHandler("postfix", "pipe", parseSmtpPayload)
	registerHandler("postfix", "local", parseSmtpPayload)
	registerHandler("postfix", "virtual", parseSmtpPayload)

	// Reported by some curious user
	registerHandler("amavis-inject", "smtp", parseSmtpPayload)
	registerHandler("postfix-slow", "smtp", parseSmtpPayload)
}

type RawSmtpSentStatus struct {
	Queue                   string
	RecipientLocalPart      string
	RecipientDomainPart     string
	OrigRecipientLocalPart  string
	OrigRecipientDomainPart string
	RelayName               string
	RelayIpOrPath           string
	RelayPort               string
	Delay                   string
	Delays                  [5]string
	Dsn                     string
	Status                  string
	ExtraMessage            string

	// parsed extra message
	ExtraMessagePayloadType              PayloadType
	ExtraMessageSmtpSentStatusSentQueued SmtpSentStatusExtraMessageSentQueued
}

type SmtpSentStatusExtraMessageSentQueued struct {
	SmtpCode string
	Dsn      string
	IP       string
	Port     string
	Queue    string
}

func parseSmtpPayload(payloadLine string) (RawPayload, error) {
	r, parsed := parseSmtpSentStatus(payloadLine)

	if !parsed {
		return RawPayload{PayloadType: PayloadTypeUnsupported}, ErrUnsupportedLogLine
	}

	// TODO: refactor this code when mode kinds of extra messages are parsed
	if extraMessage, parsed := parseSmtpSentStatusExtraMessageSentQueued(r.ExtraMessage); parsed {
		r.ExtraMessageSmtpSentStatusSentQueued = extraMessage
		r.ExtraMessagePayloadType = PayloadTypeSmtpMessageStatusSentQueued
	}

	return RawPayload{
		PayloadType:       PayloadTypeSmtpMessageStatus,
		RawSmtpSentStatus: r,
	}, nil
}
