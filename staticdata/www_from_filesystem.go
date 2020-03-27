// +build dev !release

package staticdata

import (
	"net/http"
)

var (
	HttpAssets http.FileSystem = http.Dir("./www")
)
