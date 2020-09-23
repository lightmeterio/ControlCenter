package emailutil

import (
	"regexp"
)

var (
	emailRegexp = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)
)

func IsValidEmailAddress(email string) bool {
	return emailRegexp.Match([]byte(email))
}
