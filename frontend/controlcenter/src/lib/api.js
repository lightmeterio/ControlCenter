const BASE_URL = "http://localhost:8003";

import axios from "axios";
axios.defaults.withCredentials = true;
import Vue from "vue";

export async function submitGeneralForm(data, successMessage) {
  let formData = getFormData(data);

  const response = await axios.post(
    BASE_URL + "/settings?setting=general",
    new URLSearchParams(formData)
  );

  if (response.status !== 200) {
    alert(
      Vue.prototype.$gettext("Error on saving notification settings!") +
        " " +
        response.data
    );
    return;
  }

  if (successMessage !== false) {
    alert(Vue.prototype.$gettext("Saved general settings"));
  }
  //_paq.push(["trackEvent", "SaveGeneralSettings", "success"]);
}

export function submitLoginForm(formData, callback) {
  const data = new URLSearchParams(getFormData(formData));
  axios
    .post(BASE_URL + "/login", data)
    .then(function(response) {
      if (
        response.data !== null &&
        response.data.Error !== undefined &&
        response.data.Error.length > 0
      ) {
        alert(Vue.prototype.$gettext("Error") + ":" + response.data.Error);
        return;
      }
      callback();
    })
    .catch(function(err) {
      alert(Vue.prototype.$gettext("Server Error!"));
      console.log(err);
    });
}

export async function submitNotificationsSettingsForm(data) {
  let notificationsSettingsFormData = getFormData(data);

  const response = await axios.post(
    BASE_URL + "/settings?setting=notification",
    new URLSearchParams(notificationsSettingsFormData)
  );

  if (response.status !== 200) {
    alert(
      Vue.prototype.$gettext("Error on saving notification settings!") +
        " " +
        response.data
    );
    return;
  }

  alert(Vue.prototype.$gettext("Saved notification settings"));
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
    .then(function(response) {
      if (
        response.data !== null &&
        response.data.Error !== undefined &&
        response.data.Error.length > 0
      ) {
        alert(Vue.prototype.$gettext("Error") + ":" + response.data.Error);
        return;
      }
      redirect();
    })
    .catch(function(err) {
      alert(Vue.prototype.$gettext("Server Error!"));
      console.log(err);
    });
}

export function submitRegisterForm(registrationData, settingsData, redirect) {
  let registrationFormData = getFormData(registrationData);
  let settingsFormData = getFormData(settingsData);

  axios
    .post(BASE_URL + "/register", new URLSearchParams(registrationFormData))
    .then(function(response) {
      if (
        response.data !== null &&
        response.data.Error !== undefined &&
        response.data.Error.length > 0
      ) {
        let message = (function() {
          // add hints of pwd weakness
          if (
            response.data.Detailed &&
            response.data.Detailed.Sequence &&
            response.data.Detailed.Sequence[0].pattern
          ) {
            return (
              Vue.prototype.$gettext("Error") +
              response.data.Error +
              ".\n" +
              Vue.prototype.$gettext("Vulnerable to") +
              ":" +
              response.data.Detailed.Sequence[0].pattern +
              "."
            );
          }

          return (
            alert(Vue.prototype.$gettext("Error")) + ":" + response.data.Error
          );
        })();

        alert(message);
        return;
      }

      axios
        .post(
          BASE_URL + "/settings?setting=initSetup",
          new URLSearchParams(settingsFormData)
        )
        .then(function(response) {
          if (response.status !== 200) {
            alert(
              Vue.prototype.$gettext("Settings Error on initial setup!") +
                " " +
                response.data
            );
            return;
          }
          // todo tracking
          redirect();
        })
        .catch(function(err) {
          alert(Vue.prototype.$gettext("Settings Error on initial setup!"));
          console.log(err);
        });
    })
    .catch(function(err) {
      alert(Vue.prototype.$gettext("Server Error!"));
      console.log(err);
    });
}

function getFormData(object) {
  const formData = new FormData();
  Object.keys(object).forEach(key => formData.append(key, object[key]));
  return formData;
}

export function getApplicationInfo() {
  return axios.get(BASE_URL + "/api/v0/appVersion");
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

  return axios.get(
    BASE_URL + "/api/v0/" + methodName + "?" + timeIntervalUrlParams()
  );
}
