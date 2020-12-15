// SPDX-FileCopyrightText: 2020,  Lightmeter <hello@lightmeter.io>
// SPDX-License-Identifier: AGPL-3.0

package migrations

import (
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

type Field1 struct {
	Field2 []Field2 `json:"field_2"`
}

type Field2 struct {
	Field3 string `json:"field_3"`
	Field4 Field4 `json:"field_4"`
}

type Field4 struct {
	Field5 string `json:"field_5"`
	Field6 int    `json:"field_6"`
}

type TestType struct {
	Field1 Field1 `json:"field_1"`
}

func TestFixingJsonMarshalling(t *testing.T) {
	Convey("Test Json Marshalling", t, func() {

		var oldValue map[string]interface{}
		oldEncoded := `{"Field1":{"Field2":[{"Field3":"aaa","Field4":{"Field5":"bbb","Field6":42}},{"Field3":"ccc","Field4":{"Field5":"ddd","Field6":35}}]}}`

		err := json.Unmarshal([]byte(oldEncoded), &oldValue)
		So(err, ShouldBeNil)

		nameFixup := func(s string) string {
			m := map[string]string{
				"Field1": "field_1",
				"Field2": "field_2",
				"Field3": "field_3",
				"Field4": "field_4",
				"Field5": "field_5",
				"Field6": "field_6",
			}

			return m[s]
		}

		fixedValue, err := fixKeyNames(oldValue, nameFixup)
		So(err, ShouldBeNil)

		encodedOldValue, err := json.Marshal(fixedValue)
		So(err, ShouldBeNil)

		var newValue TestType
		err = json.Unmarshal(encodedOldValue, &newValue)
		So(err, ShouldBeNil)

		So(newValue, ShouldResemble, TestType{
			Field1: Field1{
				Field2: []Field2{
					{
						Field3: "aaa",
						Field4: Field4{
							Field5: "bbb",
							Field6: 42,
						},
					},
					{
						Field3: "ccc",
						Field4: Field4{
							Field5: "ddd",
							Field6: 35,
						},
					},
				},
			},
		})

	})
}
