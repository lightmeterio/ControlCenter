// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build dev || !release
// +build dev !release

//go:generate go run github.com/golang/mock/mockgen -destination=mock/dashboard_mock.go gitlab.com/lightmeter/controlcenter/dashboard Dashboard

package dashboard
