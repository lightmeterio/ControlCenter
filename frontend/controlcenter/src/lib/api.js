// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

const BASE_URL = process.env.VUE_APP_CONTROLCENTER_BACKEND_BASE_URL;
import { trackEvent } from "@/lib/util";

import axios from "axios";
axios.defaults.withCredentials = true;
import Vue from "vue";

export function submitGeneralForm(data, successMessage) {
  let formData = getFormData(data);

  axios
    .post(BASE_URL + "settings?setting=general", new URLSearchParams(formData))
    .then(function() {
      trackEvent("SaveGeneralSettings", "success");
      if (successMessage !== false) {
        alert(Vue.prototype.$gettext("Saved general settings"));
      }
    })
    .catch(builderErrorHandler("setting_general"));
}

export function submitLoginForm(formData, callback) {
  const data = new URLSearchParams(getFormData(formData));
  axios
    .post(BASE_URL + "login", data)
    .then(function() {
      callback();
    })
    .catch(builderErrorHandler("login"));
}

export function submitNotificationsSettingsForm(data) {
  let notificationsSettingsFormData = getFormData(data);

  return axios
    .post(
      BASE_URL + "settings?setting=notification",
      new URLSearchParams(notificationsSettingsFormData)
    )
    .then(function() {
      trackEvent("SaveNotificationSettings", "success");
      alert(Vue.prototype.$gettext("Saved notification settings"));
    })
    .catch(builderErrorHandler("settings"));
}

export function getSettings() {
  return axios
    .get(BASE_URL + "settings")
    .catch(builderErrorHandler("settings"));
}

export function getMetaLanguage() {
  return axios
    .get(BASE_URL + "language/metadata")
    .catch(builderErrorHandler("language_metadata"));
}

export function getUserInfo() {
  return axios
    .get(BASE_URL + "api/v0/userInfo")
    .catch(builderErrorHandler("user_info"));
}

export function logout(redirect) {
  axios
    .post(BASE_URL + "logout", null)
    .then(function() {
      redirect();
    })
    .catch(builderErrorHandler("logout"));
}

export function submitRegisterForm(registrationData, settingsData, redirect) {
  let registrationFormData = getFormData(registrationData);
  let settingsFormData = getFormData(settingsData);

  axios
    .post(BASE_URL + "register", new URLSearchParams(registrationFormData))
    .then(function() {
      axios
        .post(
          BASE_URL + "settings?setting=initSetup",
          new URLSearchParams(settingsFormData)
        )
        .then(function() {
          trackEvent("RegisterAdmin", "success");
          if (settingsData.subscribe_newsletter) {
            trackEvent("RegisterAdmin", "newsletterOn");
          }
          trackEvent("RegisterAdmin", settingsData.email_kind);

          redirect();
        })
        .catch(builderErrorHandler("initSetup"));
    })
    .catch(function(err) {
      // add hints of pwd weakness
      if (
        err.response.data.Detailed &&
        err.response.data.Detailed.Sequence &&
        err.response.data.Detailed.Sequence[0].pattern
      ) {
        let errTranslation = Vue.prototype.$gettext("Error: %{error}.");
        let errMessage = Vue.prototype.$gettextInterpolate(errTranslation, {
          error: err.response.data.Error
        });
        let descTranslation = Vue.prototype.$gettext(
          "Vulnerable to: %{description}."
        );
        let descMessage = Vue.prototype.$gettextInterpolate(descTranslation, {
          description: err.response.data.Detailed.Sequence[0].pattern
        });

        alert(errMessage + "\n" + descMessage);

        return;
      }

      alertError(err, "register");
    });
}

function getFormData(object) {
  const formData = new FormData();
  Object.keys(object).forEach(key => formData.append(key, object[key]));
  return formData;
}

export function getApplicationInfo() {
  return axios
    .get(BASE_URL + "api/v0/appVersion")
    .catch(builderErrorHandler("app_version"));
}

export function getIsNotLoginOrNotRegistered() {
  return axios.get(BASE_URL + "auth/check");
}

export function fetchInsights(selectedDateFrom, selectedDateTo, filter, sort) {
  let formData = new FormData();

  formData.append("from", selectedDateFrom);
  formData.append("to", selectedDateTo);

  let setCategoryFilter = function(category) {
    formData.append("filter", "category");
    formData.append("category", category);
  };

  let s = filter.split("-");

  if (s.length === 2 && s[0] === "category") {
    setCategoryFilter(s[1]);
  }

  formData.append("entries", "6");
  formData.append("order", sort);

  var params = new URLSearchParams(formData);

  return axios.get(BASE_URL + "api/v0/fetchInsights?" + params.toString());
}

export function fetchGraphDataAsJsonWithTimeInterval(
  selectedDateFrom,
  selectedDateTo,
  methodName
) {
  const timeIntervalUrlParams = function() {
    return "from=" + selectedDateFrom + "&to=" + selectedDateTo;
  };

  return axios
    .get(BASE_URL + "api/v0/" + methodName + "?" + timeIntervalUrlParams())
    .catch(errorHandler);
}

function errorHandler(err) {
  alertError(err.response, null);
}

function builderErrorHandler(eventName) {
  return function(err) {
    alertError(err.response, eventName);
  };
}

function alertError(response, eventName) {
  console.log("dump response: ", response);
  let errMsg = (function() {
    if (response.data.Error !== undefined) {
      return ": " + response.data.Error;
    }

    if (response.data !== "") {
      return ": " + response.data;
    }

    return "";
  })();

  if (eventName !== null && response.statusCode >= 500) {
    trackEvent(eventName, errMsg);
  }

  let translation = Vue.prototype.$gettext("Error: %{err}");
  let message = Vue.prototype.$gettextInterpolate(translation, { err: errMsg });

  alert(message);
}
