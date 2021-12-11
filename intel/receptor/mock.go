// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build dev || !release
// +build dev !release

//go:generate go run github.com/golang/mock/mockgen -self_package gitlab.com/lightmeter/controlcenter/intel/receptor  -destination=receptor_mock.go -package receptor gitlab.com/lightmeter/controlcenter/intel/receptor Requester

package receptor
