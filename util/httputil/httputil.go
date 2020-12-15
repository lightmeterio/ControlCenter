// SPDX-FileCopyrightText: 2020,  Lightmeter <hello@lightmeter.io>
// SPDX-License-Identifier: AGPL-3.0

package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func WriteJson(w http.ResponseWriter, response interface{}, status int) error {
	encodedJson, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("Error encoding error json response %w", err)
	}

	w.Header()["Content-Type"] = []string{"application/json"}
	w.WriteHeader(status)

	_, err = w.Write(encodedJson)
	if err != nil {
		return fmt.Errorf("Error write byte buffer %w", err)
	}

	return nil
}
