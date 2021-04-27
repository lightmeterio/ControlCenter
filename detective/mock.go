// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build dev !release

//go:generate go run github.com/golang/mock/mockgen -destination=mock/detective_mock.go gitlab.com/lightmeter/controlcenter/detective Detective

package detective
