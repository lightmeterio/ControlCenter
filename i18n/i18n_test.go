// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package i18n

import (
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/po"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSettingsPage(t *testing.T) {
	Convey("Retrieve languages", t, func() {

		s := NewService(po.DefaultCatalog)

		settingsServer := httptest.NewServer(httpmiddleware.New().WithError(httpmiddleware.CustomHTTPHandler(s.LanguageMetaDataHandler)))

		c := &http.Client{}

		Convey("get keys", func() {
			r, err := c.Get(settingsServer.URL)
			So(err, ShouldBeNil)
			So(r.StatusCode, ShouldEqual, http.StatusOK)
			var body map[string]interface{}
			dec := json.NewDecoder(r.Body)
			err = dec.Decode(&body)
			So(err, ShouldBeNil)

			expected := map[string]interface{}{"languages": []interface{}{map[string]interface{}{"key": "English", "value": "en"}, map[string]interface{}{"key": "Deutsch", "value": "de"}, map[string]interface{}{"key": "Português do Brasil", "value": "pt_BR"}}}

			So(body, ShouldResemble, expected)
		})

	})
}
