// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirwatcher

import (
	"bytes"
	"github.com/dsnet/compress/bzip2"
	"github.com/rs/zerolog/log"
)

// nolint:unused,deadcode
func compressBzip2(content string) []byte {
	var buf bytes.Buffer

	w, err := bzip2.NewWriter(&buf, nil)
	if err != nil {
		log.Fatal().Msg("compressing bz2 data")
	}

	if _, err := w.Write([]byte(content)); err != nil {
		log.Fatal().Msg("compressing bz2 data")
	}

	if err := w.Close(); err != nil {
		log.Fatal().Msg("closing bz2 compressing data")
	}

	return buf.Bytes()
}

// nolint:unused,deadcode
func bzip2edDataFile(s string) fakeFileDataBytes {
	return fakeFileDataBytes{compressBzip2(s)}
}
