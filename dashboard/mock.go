// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build dev || !release
// +build dev !release

//go:generate go run github.com/golang/mock/mockgen -destination=mock/dashboard_mock.go gitlab.com/lightmeter/controlcenter/dashboard Dashboard
//go:generate go run github.com/golang/mock/mockgen -destination=mock/queryable_mock.go gitlab.com/lightmeter/controlcenter/dashboard Queryable
//go:generate go run github.com/golang/mock/mockgen -destination=mock/queryablerows_mock.go gitlab.com/lightmeter/controlcenter/dashboard QueryableRows

package dashboard
