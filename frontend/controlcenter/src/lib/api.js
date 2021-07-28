// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

const BASE_URL = process.env.VUE_APP_CONTROLCENTER_BACKEND_BASE_URL;
import { trackEvent, trackEventArray, updateMatomoEmail } from "@/lib/util";

import axios from "axios";
import { newAlertError, newAlertSuccess } from "@/lib/util.js";
axios.defaults.withCredentials = true;
import Vue from "vue";

export function getAPI(endpoint) {
  return axios.get(BASE_URL + "api/v0/" + endpoint);
}

function cap(str) {
  return str ? str.charAt(0).toUpperCase() + str.slice(1) : "";
}

export function clearSettings(section, subsection) {
  let formData = new FormData();
  formData.append("action", "clear");
  formData.append("subsection", subsection);

  return axios
    .post(
      BASE_URL + "settings?setting=" + section,
      new URLSearchParams(formData)
    )
    .then(function() {
      trackEvent(
        "Clear" + cap(subsection) + cap(section) + "Settings",
        "success"
      );
    })
    .catch(
      builderErrorHandler(
        "setting_" + (subsection ? subsection + "_" : "") + section + "_clear"
      )
    );
}

export function submitGeneralForm(data, successMessage) {
  let formData = getFormData(data);

  axios
    .post(BASE_URL + "settings?setting=general", new URLSearchParams(formData))
    .then(function() {
      trackEvent("SaveGeneralSettings", "success");
      if (successMessage !== false) {
        newAlertSuccess(Vue.prototype.$gettext("Saved general settings"));
      }
    })
    .catch(builderErrorHandler("setting_general"));
}

export function submitLoginForm(formData, callback) {
  const data = new URLSearchParams(getFormData(formData));
  axios
    .post(BASE_URL + "login", data)
    .then(function() {
      updateMatomoEmail().then(function() {
        trackEvent("Login", "success");
        callback();
      });
    })
    .catch(function(err) {
      trackEvent("Login", "error");
      builderErrorHandler("login")(err);
    });
}

export function submitNotificationsSettingsForm(data, trackingInfo) {
  let notificationsSettingsFormData = getFormData(data);

  return axios
    .post(
      BASE_URL + "settings?setting=notification",
      new URLSearchParams(notificationsSettingsFormData)
    )
    .then(function() {
      for (let i in trackingInfo) {
        trackEventArray("SaveNotificationSettings", [i, trackingInfo[i]]);
      }

      newAlertSuccess(Vue.prototype.$gettext("Saved notification settings"));
    })
    .catch(builderErrorHandler("settings"));
}

export function submitDetectiveSettingsForm(data, enabled) {
  let detectiveSettingsFormData = getFormData(data);

  return axios
    .post(
      BASE_URL + "settings?setting=detective",
      new URLSearchParams(detectiveSettingsFormData)
    )
    .then(function() {
      trackEvent("MessageDetectiveEndUsers", enabled ? "enabled" : "disabled");

      newAlertSuccess(
        Vue.prototype.$gettext("Saved message detective settings")
      );
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
          updateMatomoEmail().then(function() {
            trackEvent("RegisterAdmin", "success");
            if (settingsData.subscribe_newsletter) {
              trackEvent("RegisterAdmin", "newsletterOn");
            }
            trackEvent("RegisterAdmin", settingsData.email_kind);

            redirect();
          });
        })
        .catch(builderErrorHandler("initSetup"));
    })
    .catch(function(err) {
      // add hints of pwd weakness
      if (
        err.response.data.detailed &&
        err.response.data.detailed.Sequence &&
        err.response.data.detailed.Sequence[0].pattern
      ) {
        let errTranslation = Vue.prototype.$gettext("Error: %{error}.");
        let errMessage = Vue.prototype.$gettextInterpolate(errTranslation, {
          error: err.response.data.error
        });
        let descTranslation = Vue.prototype.$gettext(
          "Vulnerable to: %{description}."
        );
        let descMessage = Vue.prototype.$gettextInterpolate(descTranslation, {
          description: err.response.data.detailed.Sequence[0].pattern
        });

        newAlertError(errMessage + "\n" + descMessage);

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

export function getIsNotLoginAndNotEndUsersEnabled() {
  return axios.get(BASE_URL + "auth/detective");
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

  formData.append("entries", "100");
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
    if (response.data.error !== undefined) {
      return ": " + response.data.error;
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

  newAlertError(message);
}

export function requestWalkthroughCompletedStatus(completed) {
  let data = new FormData();
  data.append("completed", completed);

  var params = new URLSearchParams(data);

  return axios.post(
    BASE_URL + "settings?setting=walkthrough",
    params.toString()
  );
}

/**** Message Detective ****/

export function checkMessageDelivery(
  mail_from,
  mail_to,
  date_from,
  date_to,
  page
) {
  let formData = new FormData();
  formData.append("mail_from", mail_from);
  formData.append("mail_to", mail_to);
  formData.append("from", date_from);
  formData.append("to", date_to);
  formData.append("page", page);

  var post = axios.post(
    BASE_URL + "api/v0/checkMessageDeliveryStatus",
    new URLSearchParams(formData)
  );
  post.catch(builderErrorHandler("detective_search"));
  return post;
}

export function escalateMessage(mail_from, mail_to, date_from, date_to) {
  let formData = new FormData();
  formData.append("mail_from", mail_from);
  formData.append("mail_to", mail_to);
  formData.append("from", date_from);
  formData.append("to", date_to);

  var post = axios.post(
    BASE_URL + "api/v0/escalateMessage",
    new URLSearchParams(formData)
  );

  post.catch(builderErrorHandler("detective_escalation"));
  post.then(function() {
    trackEvent("MessageDetectiveEscalate", "escalate");
  });

  return post;
}

export function oldestAvailableTimeForMessageDetective() {
  return getAPI("oldestAvailableTimeForMessageDetective");
}

/**** User ratings of insights ****/

export function postUserRating(type, rating) {
  let formData = new FormData();
  formData.append("type", type);
  formData.append("rating", rating);

  return axios
    .post(BASE_URL + "api/v0/rateInsight", new URLSearchParams(formData))
    .then(function() {
      let ratingText = {
        0: "Bad",
        1: "Neutral",
        2: "Good"
      }[rating];
      trackEvent("InsightUserRating" + ratingText, type);
    })
    .catch(builderErrorHandler("insight_user_rating"));
}
