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
import GetTextPlugin from "vue-gettext";

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
import graphdashboard from "./components/graph-dashboard";
import mainfooter from "./components/mainfooter";
import insights from "./components/insights";
import langaugeSwitcher from "./components/langauge-switcher";

Vue.component("insights", insights);
Vue.component("graphdashboard", graphdashboard);
Vue.component("langauge-switcher", langaugeSwitcher);
Vue.component("auth-page-footer", authpagefooter);
Vue.component("mainfooter", mainfooter);
Vue.component("mainheader", mainheader);
Vue.component("panel-page", panelpage);

import translations from "./translation/translations.json";

// todo(marcel) fetch from api
Vue.use(GetTextPlugin, {
  availableLanguages: {
    en: "English",
    de: "Deutsch",
    pt_BR: "Português do Brasil"
  },
  defaultLanguage: "en",
  languageVmMixin: {
    computed: {
      currentKebabCase: function() {
        return this.current.toLowerCase().replace("_", "-");
      }
    }
  },
  translations: translations,
  silent: true
});

Vue.config.productionTip = false;

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount("#app");
