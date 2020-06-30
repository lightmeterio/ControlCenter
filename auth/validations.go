package auth

import (
	"errors"
	"github.com/trustelem/zxcvbn"
	"log"
	"regexp"
	"strings"
)

var (
	ErrInvalidEmail         = errors.New("Invalid Email Address")
	ErrEmailAddressNotFound = errors.New("Email Address Not Found")
	ErrWeakPassword         = errors.New("Weak Password")
	ErrInvalidName          = errors.New("Invalid Name")
)

var (
	emailRegexp = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)
)

func validateEmail(email string) error {
	if !emailRegexp.Match([]byte(email)) {
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

	log.Println("Requested to register password with strength score:", strength.Score, "and calc time:", strength.CalcTime)

	if strength.Score < 3 {
		log.Println("Registration request denied due weak password")
		return &PasswordValidationError{err: ErrWeakPassword, Result: PasswordErrorDetailedDescription(strength)}
	}

	return nil
}
