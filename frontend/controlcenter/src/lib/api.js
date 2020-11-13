const BASE_URL = "http://localhost:8003";

export function submitLoginForm(formData) {
  const data = new URLSearchParams(getFormData(formData));

  fetch(BASE_URL + "/login", { method: "post", body: data })
    .then(res => res.json())
    .then(function(data) {
      if (data == null) {
        alert("Server Error!"); // todo "{{translate `Server Error!`}}"
        return;
      }

      if (data.Error.length > 0) {
        alert("Error: " + data.Error); // todo "{{translate `Error` }}"
        return;
      }
    })
    .catch(function(err) {
      alert("Server Error!"); // todo "{{translate `Server Error!`}}"
      console.log(err);
    });
}

export function submitRegisterForm(registrationData, settingsData) {
  let registrationFormData = getFormData(registrationData);
  let settingsFormData = getFormData(settingsData);

  fetch(BASE_URL + "/register", {
    method: "post",
    body: new URLSearchParams(registrationFormData)
  })
    .then(res => res.json())
    .then(function(data) {
      if (data == null) {
        alert("Server Error!"); // todo "{{ translate `Server Error!` }}"
        return;
      }

      if (data.Error.length > 0) {
        let message = (function() {
          // add hints of pwd weakness
          if (
            data.Detailed &&
            data.Detailed.Sequence &&
            data.Detailed.Sequence[0].pattern
          ) {
            return (
              "Error" +
              data.Error +
              ".\n" +
              "Vulnerable to: " +
              data.Detailed.Sequence[0].pattern +
              "."
            ); //"{{ translate `Error` }}: " + data.Error + ".\n" + "{{ translate `Vulnerable to` }}: " + data.Detailed.Sequence[0].pattern + '.'
          }

          return "Error:" + data.Error; //todo "{{ translate `Error` }}:" + data.Error
        })();

        alert(message);
        return;
      }

      fetch(BASE_URL + "/settings?setting=initSetup", {
        method: "post",
        body: new URLSearchParams(settingsFormData)
      })
        .then(function(data) {
          console.log(data);
          // todo tracking
          // todo forward
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
