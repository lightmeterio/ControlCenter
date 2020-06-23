package auth

import (
	"errors"
	"github.com/trustelem/zxcvbn"
	"log"
	"regexp"
)

var (
	ErrEmptyPassword = errors.New("Empty Password")
	ErrInvalidEmail  = errors.New("Invalid Email Address")
	ErrWeakPassword  = errors.New("Weak Password")
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

func validatePassword(email, password string) error {
	if len(password) == 0 {
		return ErrEmptyPassword
	}

	strength := zxcvbn.PasswordStrength(password, []string{email})

	log.Println("Requested to register password with strength score:", strength.Score, "and calc time:", strength.CalcTime)

	if strength.Score < 3 {
		log.Println("Registration request denied due weak password")
		return ErrWeakPassword
	}

	return nil
}
