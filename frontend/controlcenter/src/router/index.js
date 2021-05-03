// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

import Vue from "vue";
import VueRouter from "vue-router";
import login from "../views/login.vue";
import register from "../views/register.vue";
import settingspage from "../views/settingspage.vue";
import index from "../views/index.vue";
import admindetective from "../views/admindetective.vue";
import enduserdetective from "../views/enduserdetective.vue";
import {
  getIsNotLoginOrNotRegistered,
  getIsNotLoginAndNotEndUsersEnabled
} from "@/lib/api";

Vue.use(VueRouter);

const routes = [
  {
    path: "/",
    name: "index",
    component: index
  },
  {
    path: "/login",
    name: "login",
    component: login
  },
  {
    path: "/register",
    name: "register",
    component: register
  },
  {
    path: "/settings",
    name: "settings",
    component: settingspage
  },
  {
    path: "/detective",
    name: "detective",
    component: admindetective
  },
  {
    path: "/searchmessage",
    name: "searchmessage",
    component: enduserdetective
  },
  {
    path: "/insight-card/:id",
    name: "insight-card",
    component: index
  },
  {
    path: "*",
    name: "any",
    redirect: { name: "index" }
  }
];

const router = new VueRouter({
  routes
});

router.beforeEach((to, from, next) => {
  let customPageTitles = {
    settings: Vue.prototype.$gettext("Settings - %{mainPageTitle}"),
    register: Vue.prototype.$gettext("Registration - %{mainPageTitle}"),
    login: Vue.prototype.$gettext("Login - %{mainPageTitle}"),
    detective: Vue.prototype.$gettext("Message Detective - %{mainPageTitle}"),
    searchmessage: Vue.prototype.$gettext(
      "Search for messages - %{mainPageTitle}"
    )
  };

  let mainTitle = "Lightmeter";
  let extraText = customPageTitles[to.name];

  if (extraText !== undefined) {
    mainTitle = Vue.prototype.$gettextInterpolate(extraText, {
      mainPageTitle: mainTitle
    });
  }

  document.title = mainTitle;

  next();
});

router.beforeEach((to, from, next) => {
  if (to.name === "login") {
    getIsNotLoginOrNotRegistered()
      .then(function() {
        next({ name: "index" });
      })
      .catch(function(error) {
        if (error.response.status === 401) {
          next();
          return;
        } else if (error.response.status === 403) {
          next({ name: "register" });
          return;
        }
        return error;
      });
    return;
  }
  if (to.name === "register") {
    getIsNotLoginOrNotRegistered()
      .then(function() {
        next({ name: "index" });
      })
      .catch(function(error) {
        if (error.response.status === 401) {
          next({ name: "login" });
          return;
        } else if (error.response.status === 403) {
          next();
          return;
        }
        return error;
      });
    return;
  }
  if (to.name === "searchmessage") {
    getIsNotLoginAndNotEndUsersEnabled()
      .then(function() {
        next();
      })
      .catch(function(error) {
        return error;
      });
    return;
  }

  getIsNotLoginOrNotRegistered()
    .then(function() {
      next();
    })
    .catch(function(error) {
      if (error.response.status === 401) {
        next({
          name: "login"
        });
      } else if (error.response.status === 403) {
        next({
          name: "register"
        });
      }
      return error;
    });
});

export default router;
