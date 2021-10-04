// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build dev || !release
// +build dev !release

package workspace

// NOTE: this URL does not need to exist, but you can use it to test the reports locally
const IntelReportDestinationURL = "http://localhost:9999/reports"
