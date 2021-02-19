// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrationutil

// Given an object, if it has any maps[string]interface{} in it,
// Returns a new object with the key values transformed using a custom function
// NOTE: It does not handle pointer cycles
func FixKeyNames(o interface{}, fixup func(string) string) (interface{}, error) {
	if asMap, ok := o.(map[string]interface{}); ok {
		newMap := map[string]interface{}{}

		for k, v := range asMap {
			fixedKey := fixup(k)

			fixedValue, err := FixKeyNames(v, fixup)

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
			newValue, err := FixKeyNames(v, fixup)

			if err != nil {
				return nil, err
			}

			newSlice = append(newSlice, newValue)
		}

		return newSlice, nil
	}

	return o, nil
}
