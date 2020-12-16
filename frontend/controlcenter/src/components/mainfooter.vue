<template>

  <footer class="mt-auto">
    <div class="container">
      <div class="row justify-content-between">

        <div class="col-md-6 mt-md-0 mt-3 align-left">
          <a href="https://lightmeter.io/about/" title='About Lightmeter' target="_blank"><translate>Thank you for using Lightmeter</translate></a>. &copy; {{ year }}.
          <span class="link"> <a href="https://lightmeter.io/privacy-policy/" title='Read policy' target="_blank"><translate>Privacy Policy</translate></a></span>
          <span class="link"> <a href="mailto:hello@lightmeter.io?subject=Feedback%20on%20Lightmeter%20Control%20Center" title='What would you improve?' v-on:click="trackClick('Feedback', 'clickMailTo')" target="_blank"><translate>Feedback</translate></a></span>
        </div>

        <div class="col-md-2 mb-md-0 mb-2 align-right">
          <langauge-switcher></langauge-switcher>
        </div>

      </div>
    </div>
  </footer>

</template>
<script>
import tracking from "../mixin/global_shared.js";
export default {
  name: "mainfooter",
  mixins: [tracking],
  data() {
    return {
      year: null
    };
  },
  mounted() {
    this.year = new Date().getFullYear();
  }
};
</script>
<style>
footer {
  padding: 0.5rem 0;
}
footer .btn, footer .btn-secondary {
  padding: 0;
  margin: 0;
  font-size: 1em;
  background-color: inherit;
  border: none;
}
footer .link {
  padding-left: 0.8em;
}
footer .link::before {
  content: "\2022\00a0"; /* add bullet and space */
}
</style>
