// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httputil

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func WriteJson(w http.ResponseWriter, response interface{}, status int) (err error) {
	// use a 4KB buffer to improve write throughput, reducing the number of write() calls in the final socket
	bufferedWriter := bufio.NewWriterSize(w, 4096)

	defer errorutil.UpdateErrorFromCall(bufferedWriter.Flush, &err)

	w.Header()["Content-Type"] = []string{"application/json"}
	w.WriteHeader(status)

	encoder := json.NewEncoder(bufferedWriter)

	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("Error write byte buffer %w", err)
	}

	return nil
}
