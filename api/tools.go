// +build tools

package api

import _ "github.com/swaggo/swag/cmd/swag"

// NOTE: added to prevent `go mod tidy` of removing it as a dependency of swag
// But this is a ugly hack this package is required during dev builds!
import _ "github.com/alecthomas/template"
