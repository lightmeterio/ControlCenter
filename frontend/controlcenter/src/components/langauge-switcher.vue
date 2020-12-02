<template>
  <form id="languageForm">
    <b-dropdown
      id="dropdown-1"
      class="m-md-2"
      name="language"
      v-bind:text="getLangaugeLabel"
      v-model="$language.current"
    >
      <b-dropdown-item
        v-for="(language, key) in $language.available"
        v-bind:key="key"
        v-on:click="onSwitchLanguage(key)"
        >{{ language }}</b-dropdown-item
      >
    </b-dropdown>
  </form>
</template>
<script>
import Vue from "vue";
import { getSettings, submitGeneralForm } from "@/lib/api";
import { mapActions } from "vuex";
import tracking from "../mixin/global_shared.js";
export default {
  name: "langauge-switcher",
  mixins: [tracking],
  data() {
    return {};
  },
  computed: {
    getLangaugeLabel() {
      let label = "";
      for (const [key, value] of Object.entries(this.$language.available)) {
        if (key === Vue.config.language) {
          label = value;
        }
      }
      return label;
    }
  },
  methods: {
    onSwitchLanguage(key) {
      this.trackEvent("SwitchLanguage", key);

      Vue.config.language = key;
      this.setLanguageAction(key);

      if (this.$route.name !== "login" && this.$route.name !== "register") {
        let data = {
          app_language: key
        };
        submitGeneralForm(data, false);
      }
    },
    ...mapActions(["setLanguageAction"])
  },
  mounted() {
    if (this.$route.name !== "login" && this.$route.name !== "register") {
      getSettings().then(function(response) {
        if (
          response.data !== null &&
          response.data["general"].app_language != null
        ) {
          Vue.config.language = response.data["general"].app_language;
        } else {
          // set to fallback language
          Vue.config.language = "en";
        }
      });
    } else {
      this.setLanguageAction("en");
    }
  }
};
</script>
<style></style>
