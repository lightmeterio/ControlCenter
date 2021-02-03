// SPDX-FileCopyrightText: 2020,  Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package migrations

// Given an object, if it has any maps[string]interface{} in it,
// Returns a new object with the key values transformed using a custom function
// NOTE: It does not handle pointer cycles
func fixKeyNames(o interface{}, fixup func(string) string) (interface{}, error) {
	if asMap, ok := o.(map[string]interface{}); ok {
		newMap := map[string]interface{}{}

		for k, v := range asMap {
			fixedKey := fixup(k)

			fixedValue, err := fixKeyNames(v, fixup)

			if err != nil {
				return nil, err
			}

			newMap[fixedKey] = fixedValue
		}

		return newMap, nil
	}

	if asSlice, ok := o.([]interface{}); ok {
		newSlice := []interface{}{}

		for _, v := range asSlice {
			newValue, err := fixKeyNames(v, fixup)

			if err != nil {
				return nil, err
			}

			newSlice = append(newSlice, newValue)
		}

		return newSlice, nil
	}

	return o, nil
}
