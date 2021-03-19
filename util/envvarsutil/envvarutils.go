// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package envvarsutil

import (
	"log"
	"strconv"
)

func LookupEnvOrString(key string, defaultVal string, loopkupenv func(string) (string, bool)) string {
	if val, ok := loopkupenv(key); ok {
		return val
	}

	return defaultVal
}

func LookupEnvOrInt(key string, defaultVal int, loopkupenv func(string) (string, bool)) int {
	if val, ok := loopkupenv(key); ok {
		v, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalf("LookupEnvOrInt[%s]: %v", key, err)
		}

		return v
	}

	return defaultVal
}

func LookupEnvOrBool(key string, defaultVal bool, loopkupenv func(string) (string, bool)) bool {
	if val, ok := loopkupenv(key); ok {
		v, err := strconv.ParseBool(val)
		if err != nil {
			log.Fatalf("LookupEnvOrBool[%s]: %v", key, err)
		}

		return v
	}

	return defaultVal
}
