import Vue from "vue";
import VueRouter from "vue-router";
import login from "../views/login.vue";
import register from "../views/register.vue";
import settingspage from "../views/settingspage.vue";

Vue.use(VueRouter);

const routes = [
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

export default router;
