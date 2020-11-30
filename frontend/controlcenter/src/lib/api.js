const BASE_URL = "http://localhost:8003";

import axios from "axios";
axios.defaults.withCredentials = true;
import Vue from "vue";

export function submitGeneralForm(data, successMessage) {
  let formData = getFormData(data);

  axios
    .post(BASE_URL + "/settings?setting=general", new URLSearchParams(formData))
    .then(function() {
      if (successMessage !== false) {
        alert(Vue.prototype.$gettext("Saved general settings"));
      }
    })
    .catch(errorHandler);

  //_paq.push(["trackEvent", "SaveGeneralSettings", "success"]);
}

export function submitLoginForm(formData, callback) {
  const data = new URLSearchParams(getFormData(formData));
  axios
    .post(BASE_URL + "/login", data)
    .then(function() {
      callback();
    })
    .catch(errorHandler);
}

export function submitNotificationsSettingsForm(data) {
  let notificationsSettingsFormData = getFormData(data);

  return axios
    .post(
      BASE_URL + "/settings?setting=notification",
      new URLSearchParams(notificationsSettingsFormData)
    )
    .then(function() {
      alert(Vue.prototype.$gettext("Saved notification settings"));
    })
    .catch(errorHandler);
  //todo _paq.push(["trackEvent", "SaveNotificationSettings", "success"]);
}

export function getSettings() {
  return axios.get(BASE_URL + "/settings");
}

export function getMetaLanguage() {
  return axios.get(BASE_URL + "/language/metadata");
}

export function logout(redirect) {
  axios
    .post(BASE_URL + "/logout", null)
    .then(function() {
      redirect();
    })
    .catch(errorHandler);
}

export function submitRegisterForm(registrationData, settingsData, redirect) {
  let registrationFormData = getFormData(registrationData);
  let settingsFormData = getFormData(settingsData);

  axios
    .post(BASE_URL + "/register", new URLSearchParams(registrationFormData))
    .then(function() {
      axios
        .post(
          BASE_URL + "/settings?setting=initSetup",
          new URLSearchParams(settingsFormData)
        )
        .then(function() {
          // todo tracking
          redirect();
        })
        .catch(errorHandler);
    })
    .catch(function(err) {
      // add hints of pwd weakness
      if (
        err.response.data.Detailed &&
        err.response.data.Detailed.Sequence &&
        err.response.data.Detailed.Sequence[0].pattern
      ) {
        alert(
          Vue.prototype.$gettext("Error") +
            err.response.data.Error +
            ".\n" +
            Vue.prototype.$gettext("Vulnerable to: ") +
            err.response.data.Detailed.Sequence[0].pattern +
            "."
        );
        return;
      }
      alertError(err);
    });
}

function getFormData(object) {
  const formData = new FormData();
  Object.keys(object).forEach(key => formData.append(key, object[key]));
  return formData;
}

export function getApplicationInfo() {
  return axios.get(BASE_URL + "/api/v0/appVersion").catch(errorHandler);
}

export function getIsNotLoginOrNotRegistered() {
  return axios.get(BASE_URL + "/auth/check");
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

  return axios.get(BASE_URL + "/api/v0/fetchInsights?" + params.toString());
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
    .get(BASE_URL + "/api/v0/" + methodName + "?" + timeIntervalUrlParams())
    .catch(errorHandler);
}

function errorHandler(err) {
  alertError(err.response);
}

function alertError(response) {
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

  alert(Vue.prototype.$gettext("Error") + errMsg);
}
