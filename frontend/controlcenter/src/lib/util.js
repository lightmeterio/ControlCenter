// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { getUserInfo, getApplicationInfo } from "@/lib/api.js";
import posthog from "posthog-js";

posthog.init("phc_WuCKJ0vL2fNge72as4MarP5Y5w13xn3FOZAisy9ofnN", {
  api_host: "https://posthog.lightmeter.io"
});

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
  posthog.capture(eventName, { property: value });
  window._paq.push(["trackEvent", eventName, value]);
}

export function trackClick(eventName, value) {
  posthog.capture(eventName, { property: value });
  window._paq.push(["trackEvent", eventName, value]);
}

export function updateMatomoEmail() {
  return getUserInfo().then(function(userInfo) {
    getApplicationInfo().then(function(appInfo) {
      posthog.identify(userInfo.data.instance_id, {
        email: userInfo.data.user.email,
        appVersion: appInfo.data.version
      });
    });
    window._paq.push(["setCustomDimension", 2, userInfo.data.user.email]);
  });
}

// NOTE: This is not ideal and code should rather use their context vue wherever possible
function getStore() {
  if (
    !document.getElementById("app") ||
    !document.getElementById("app").__vue__ ||
    !document.getElementById("app").__vue__.$store
  )
    return false;

  return document.getElementById("app").__vue__.$store;
}

export function newAlertError(message) {
  let store = getStore();
  store ? store.dispatch("newAlertError", message) : alert(message);
}

export function newAlertSuccess(message) {
  let store = getStore();
  store ? store.dispatch("newAlertSuccess", message) : alert(message);
}
