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

function getFormData(object) {
  const formData = new FormData();
  Object.keys(object).forEach(key => formData.append(key, object[key]));
  return formData;
}
