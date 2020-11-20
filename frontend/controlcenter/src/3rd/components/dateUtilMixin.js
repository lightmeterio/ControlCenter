import { getDateUtil } from "./util.js";

export default {
  props: {
    dateUtil: {
      type: [Object, String],
      default: "native"
    }
  },
  created() {
    this.$dateUtil = getDateUtil("native");
  }
};
