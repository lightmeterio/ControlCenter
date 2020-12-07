import { getIsNotLoginOrNotRegistered } from "../lib/api.js";

export default {
  methods: {
    ValidSessionCheck: function() {
      let vue = this;
      let s = setInterval(function() {
        getIsNotLoginOrNotRegistered().catch(function() {
          vue.$router.push({ name: "login" });
        });
      }, 5000);
      return s;
    }
  }
};
