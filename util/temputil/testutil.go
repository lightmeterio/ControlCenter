// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package temputil

import (
	"io/ioutil"
	"os"
	"testing"
)

func TempDir(t *testing.T) (string, func()) {
	dir, e := ioutil.TempDir("", "lightmeter-tests-*")

	if e != nil {
		panic("error creating temp dir")
	}

	return dir, func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal("Could not remove tempdir", dir, "error:", err)
		}
	}
}
