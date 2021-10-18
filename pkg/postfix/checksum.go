// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postfix

import (
	"hash"
	"hash/fnv"
)

// Sum is the checksum for a record
type Sum int64

// NOTE: Hasher is not thread safe!
type Hasher struct {
	// NOTE: we use a 32-bit hash function as we need something that fits into a int64 value.
	// as it's stored in a sqlite database, which has only int64 support.
	// we could use Hash64 (which uses uint64) and flip some bits to turn it into a int64,
	// but for now, for the sake of simplicity, 32-bit should do the job.
	// TODO: benchmark 32-bit vs 64-bit as this can be critical for performance.
	h hash.Hash32
}

func NewHasher() *Hasher {
	return &Hasher{h: fnv.New32a()}
}

func ComputeChecksum(h *Hasher, r Record) Sum {
	h.h.Reset()
	_, _ = h.h.Write([]byte(r.Line))
	hashValue := int64(h.h.Sum32())

	return Sum(hashValue)
}
