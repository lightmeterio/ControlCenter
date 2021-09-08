// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build dev || !release
// +build dev !release

//go:generate go run github.com/golang/mock/mockgen -destination=mock/fetcher_mock.go gitlab.com/lightmeter/controlcenter/insights/core Fetcher
//go:generate go run github.com/golang/mock/mockgen -destination=mock/progress_mock.go gitlab.com/lightmeter/controlcenter/insights/core ProgressFetcher

package core
