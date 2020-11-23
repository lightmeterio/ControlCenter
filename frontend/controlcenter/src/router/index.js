import Vue from "vue";
import VueRouter from "vue-router";
import login from "../views/login.vue";
import register from "../views/register.vue";
import settingspage from "../views/settingspage.vue";
import index from "../views/index.vue";
import { getIsNotLoginOrNotRegistered } from "@/lib/api";

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
  }
];

const router = new VueRouter({
  routes
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
