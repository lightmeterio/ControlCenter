// +build dev

package staticdata

import (
	"net/http"
)

var (
	HttpAssets http.FileSystem = http.Dir("./www")
)
