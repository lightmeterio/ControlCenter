// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"errors"
	"github.com/rs/zerolog/log"
	"github.com/trustelem/zxcvbn"
	"gitlab.com/lightmeter/controlcenter/util/emailutil"
	"strings"
)

var (
	ErrInvalidEmail         = errors.New("Invalid Email Address")
	ErrEmailAddressNotFound = errors.New("Email Address Not Found")
	ErrWeakPassword         = errors.New("Weak Password")
	ErrInvalidName          = errors.New("Invalid Name")
)

func validateEmail(email string) error {
	if !emailutil.IsValidEmailAddress(email) {
		return ErrInvalidEmail
	}

	return nil
}

func validateName(name string) error {
	if len(strings.TrimSpace(name)) == 0 {
		return ErrInvalidName
	}

	return nil
}

type PasswordErrorDetailedDescription zxcvbn.Result

type PasswordValidationError struct {
	err    error
	Result PasswordErrorDetailedDescription
}

func (e *PasswordValidationError) Unwrap() error {
	return e.err
}

func (e *PasswordValidationError) Error() string {
	return e.err.Error()
}

func validatePassword(email, name, password string) error {
	strength := zxcvbn.PasswordStrength(password, []string{email, name})

	log.Info().Msgf("Requested to register password with strength score: %d and calc time: %f", strength.Score, strength.CalcTime)

	if strength.Score < 3 {
		log.Info().Msg("Registration request denied due weak password")
		return &PasswordValidationError{err: ErrWeakPassword, Result: PasswordErrorDetailedDescription(strength)}
	}

	return nil
}
