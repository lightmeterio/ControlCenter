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
import VueMatomo from "vue-matomo";

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

import { getApplicationInfo } from "./lib/api.js";

getApplicationInfo().then(function(response) {
  Vue.use(VueMatomo, {
    host: "https://matomo.lightmeter.io/",
    siteId: 3,
    trackerFileName: "matomo",

    // Overrides the autogenerated tracker endpoint entirely
    // Default: undefined
    // trackerUrl: 'https://example.com/whatever/endpoint/you/have',

    // Overrides the autogenerated tracker script path entirely
    // Default: undefined
    // trackerScriptUrl: 'https://example.com/whatever/script/path/you/have',

    // Enables automatically registering pageviews on the router
    router: router,

    // Enables link tracking on regular links. Note that this won't
    // work for routing links (ie. internal Vue router links)
    // Default: true
    enableLinkTracking: true,

    // Require consent before sending tracking information to matomo
    // Default: false
    requireConsent: false,

    // Whether to track the initial page view
    // Default: true
    trackInitialView: true,

    // Run Matomo without cookies
    // Default: false
    disableCookies: false,

    // Enable the heartbeat timer (https://developer.matomo.org/guides/tracking-javascript-guide#accurately-measure-the-time-spent-on-each-page)
    // Default: false
    enableHeartBeatTimer: true,

    // Set the heartbeat timer interval
    // Default: 15
    heartBeatTimerInterval: 15,

    // Whether or not to log debug information
    // Default: false
    debug: false,

    // UserID passed to Matomo (see https://developer.matomo.org/guides/tracking-javascript-guide#user-id)
    // Default: undefined
    userId: undefined,

    // Share the tracking cookie across subdomains (see https://developer.matomo.org/guides/tracking-javascript-guide#measuring-domains-andor-sub-domains)
    // Default: undefined, example '*.example.com'
    cookieDomain: undefined,

    // Tell Matomo the website domain so that clicks on these domains are not tracked as 'Outlinks'
    // Default: undefined, example: '*.example.com'
    domains: undefined,

    // A list of pre-initialization actions that run before matomo is loaded
    // Default: []
    // Example: [
    //   ['API_method_name', parameter_list],
    //   ['setCustomVariable','1','VisitorType','Member'],
    //   ['appendToTrackingUrl', 'new_visit=1'],
    //   etc.
    // ]
    preInitActions: [["setCustomDimension", 1, response.data.version]]
  });
});

Vue.config.productionTip = false;

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount("#app");