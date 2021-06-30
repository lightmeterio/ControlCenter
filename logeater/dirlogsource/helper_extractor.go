// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirlogsource

import (
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"io"
)

//nolint:deadcode,unused
func extractTarGz(r io.Reader, outDir string) {
	testutil.ExtractTarGz(r, outDir)
}
