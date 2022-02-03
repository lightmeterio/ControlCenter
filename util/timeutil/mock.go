// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:generate go run github.com/golang/mock/mockgen -self_package gitlab.com/lightmeter/controlcenter/util/timeutil  -destination=timeutil_mock.go -package timeutil gitlab.com/lightmeter/controlcenter/util/timeutil Clock

package timeutil
