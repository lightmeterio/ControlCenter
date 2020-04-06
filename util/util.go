package util

import "log"

func MustSucceed(err error, msg string) {
	if err != nil {
		log.Fatal("MustSucceed:", msg, "error:", err)
	}
}
