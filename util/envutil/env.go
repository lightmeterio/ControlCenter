// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package envutil

import (
	"fmt"
	"strconv"
)

func LookupEnvOrString(key string, defaultVal string, loopkupenv func(string) (string, bool)) string {
	if val, ok := loopkupenv(key); ok {
		return val
	}

	return defaultVal
}

func LookupEnvOrBool(key string, defaultVal bool, loopkupenv func(string) (string, bool)) (bool, error) {
	if val, ok := loopkupenv(key); ok {
		v, err := strconv.ParseBool(val)
		if err != nil {
			return v, fmt.Errorf("Boolean env var %v boolean value could not be parsed: %w", key, err)
		}

		return v, nil
	}

	return defaultVal, nil
}

func LookupEnvOrInt(key string, defaultVal int64, loopkupenv func(string) (string, bool)) (int64, error) {
	if val, ok := loopkupenv(key); ok {
		v, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return v, fmt.Errorf("Integer env var %v integer value could not be parsed: %w", key, err)
		}

		return v, nil
	}

	return defaultVal, nil
}
