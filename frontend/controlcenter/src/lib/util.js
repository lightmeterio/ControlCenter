// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

export function togglePasswordShow() {
  let attrValue = document
    .querySelector("#show_hide_password input")
    .getAttribute("type");
  if (attrValue === "text") {
    document
      .querySelector("#show_hide_password input")
      .setAttribute("type", "password");
    let i = document.querySelector("#show_hide_password svg");
    i.classList.add("fa-eye");
    i.classList.remove("fa-eye-slash");
  } else if (attrValue === "password") {
    document
      .querySelector("#show_hide_password input")
      .setAttribute("type", "text");
    let i = document.querySelector("#show_hide_password svg");
    i.classList.remove("fa-eye");
    i.classList.add("fa-eye-slash");
  }
}

export function trackEvent(eventName, value) {
  window._paq.push(["trackEvent", eventName, value]);
}

export function trackEventArray(eventName, value) {
  window._paq.push(["trackEvent", eventName, value].concat(value));
}

export function trackCLick(eventName, value) {
  window._paq.push(["trackEvent", eventName, value]);
}
