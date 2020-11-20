const BASE_URL = "http://localhost:8003";

import axios from "axios";
axios.defaults.withCredentials = true;

export async function submitGeneralForm(data) {
  let formData = getFormData(data);

  const response = await axios.post(
    BASE_URL + "/settings?setting=general",
    new URLSearchParams(formData)
  );

  if (response.status !== 200) {
    alert("Error on saving notification settings!" + " " + response.data); //todo  "{{ translate `Error on saving notification settings!` }}"
    return;
  }

  alert("Saved general settings");
  //_paq.push(["trackEvent", "SaveGeneralSettings", "success"]);
}

export function submitLoginForm(formData, redirect) {
  const data = new URLSearchParams(getFormData(formData));
  axios
    .post(BASE_URL + "/login", data)
    .then(function(response) {
      if (
        response.data !== null &&
        response.data.Error !== undefined &&
        response.data.Error.length > 0
      ) {
        alert("Error: " + response.data.Error); // todo "{{translate `Error` }}"
        return;
      }
      redirect();
    })
    .catch(function(err) {
      alert("Server Error!"); // todo "{{translate `Server Error!`}}"
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
    alert("Error on saving notification settings!" + " " + response.data); //todo  "{{ translate `Error on saving notification settings!` }}"
    return;
  }

  alert("Saved notification settings"); // todo {{ translate `Saved notification settings` }}
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
        alert("Error: " + response.data.Error); // todo "{{translate `Error` }}"
        return;
      }
      redirect();
    })
    .catch(function(err) {
      alert("Server Error!"); // todo "{{translate `Server Error!`}}"
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
              "Error" +
              response.data.Error +
              ".\n" +
              "Vulnerable to: " +
              response.data.Detailed.Sequence[0].pattern +
              "."
            ); //todo "{{ translate `Error` }}: " + data.Error + ".\n" + "{{ translate `Vulnerable to` }}: " + data.Detailed.Sequence[0].pattern + '.'
          }

          return "Error:" + response.data.Error; //todo "{{ translate `Error` }}:" + data.Error
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
            alert("Settings Error on initial setup!" + " " + response.data); //todo  "{{ translate `Error Settings Error on initial setup!` }}"
            return;
          }
          // todo tracking
          redirect();
        })
        .catch(function(err) {
          alert("Settings Error on initial setup!"); // todo {{ translate `Settings Error on initial setup!` }}
          console.log(err);
        });
    })
    .catch(function(err) {
      alert("Server Error!"); //todo "{{ translate `Server Error!` }}"
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
