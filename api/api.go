package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// @title Lightmeter ControlCenter HTTP API
// @version 0.1
// @description API for user interfaces
// @contact.name Lightmeter Team
// @contact.url http://lightmeter.io
// @contact.email dev@lightmeter.io
// @license.name GNU Affero General Public License 3.0
// @license.url https://www.gnu.org/licenses/agpl-3.0.en.html

func serveJson(w http.ResponseWriter, r *http.Request, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")

	encoded, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("Encoding as JSON in the http API: %w", err)
	}

	_, err = w.Write(encoded)
	if err != nil {
		return err
	}

	return nil
}
