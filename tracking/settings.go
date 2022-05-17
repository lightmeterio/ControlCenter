// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

const SettingsKey = `tracking`

type Settings struct {
	Filters FiltersDescription `json:"filters"`
}
