import Vue from "vue";
import App from "./App.vue";
import router from "./router";
import store from "./store";

/*
  Import of third party libs
 */
import { BootstrapVue, BootstrapVueIcons } from "bootstrap-vue";
Vue.use(BootstrapVue);
Vue.use(BootstrapVueIcons);

/*
  Import of third party css and javascript
 */

import "bootstrap/dist/css/bootstrap.css";
import "bootstrap-vue/dist/bootstrap-vue.css";
import "@fortawesome/fontawesome-free/css/all.css";
import "./assets/css/panel-page.css";

import "@fortawesome/fontawesome-free/js/all.js";
/*
  Import components
 */

import authpagefooter from "./components/auth-page-footer";
import panelpage from "./components/panelpage";
import mainheader from "./components/mainheader";
import mainfooter from "./components/mainfooter";
import insights from "./components/insights";

Vue.component("insights", insights);
Vue.component("auth-page-footer", authpagefooter);
Vue.component("mainfooter", mainfooter);
Vue.component("mainheader", mainheader);
Vue.component("panel-page", panelpage);

Vue.config.productionTip = false;

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount("#app");
