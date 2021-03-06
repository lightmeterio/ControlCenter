// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"net"

	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser"
)

func init() {
	registerHandler(rawparser.PayloadTypeDovecotAuthFailedWithReason, convertDovecotAuthFailed)
}

type DovecotAuthFailed struct {
	DB                string
	Username          string
	IP                net.IP
	Reason            DovecotAuthFailedReason
	ReasonExplanation string
}

type DovecotAuthFailedReason int

var dovecotAuthFailedReasonAsStrings = map[DovecotAuthFailedReason]string{
	DovecotAuthFailedReasonPasswordMismatch:  "Password Mismatch",
	DovecotAuthFailedReasonAuthPolicyRefusal: "Policy Server Refusal",
	DovecotAuthFailedReasonUnknownUser:       "Unknown User",
}

func (r DovecotAuthFailedReason) String() string {
	return dovecotAuthFailedReasonAsStrings[r]
}

const (
	// TODO: Should we use iota for those values?
	// If we do not store them in a database, iota works just well
	UnsupportedDovecotAuthFailedReason       DovecotAuthFailedReason = 0
	DovecotAuthFailedReasonPasswordMismatch  DovecotAuthFailedReason = 1
	DovecotAuthFailedReasonUnknownUser       DovecotAuthFailedReason = 2
	DovecotAuthFailedReasonAuthPolicyRefusal DovecotAuthFailedReason = 3
)

func (DovecotAuthFailed) isPayload() {
	// required by Payload interface
}

func convertDovecotAuthFailed(r rawparser.RawPayload) (Payload, error) {
	p := r.DovecotAuthFailedWithReason

	ip, err := parseIP(p.IP)
	if err != nil {
		return nil, err
	}

	reason := func() DovecotAuthFailedReason {
		if len(p.DovecotAuthFailedReasonPasswordMismatch) > 0 {
			return DovecotAuthFailedReasonPasswordMismatch
		}

		if len(p.DovecotAuthFailedReasonUnknownUser) > 0 {
			return DovecotAuthFailedReasonUnknownUser
		}

		if len(p.DovecotAuthFailedReasonAuthPolicyRefusal) > 0 {
			return DovecotAuthFailedReasonAuthPolicyRefusal
		}

		return UnsupportedDovecotAuthFailedReason
	}()

	return DovecotAuthFailed{
		DB:                p.DB,
		Username:          p.Username,
		IP:                ip,
		Reason:            reason,
		ReasonExplanation: p.ReasonExplanation,
	}, nil
}
