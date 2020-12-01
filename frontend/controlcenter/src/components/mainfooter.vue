<template>
  <footer id="normal-footer" class="mt-auto">
    <div class="container d-flex">
      <p class="thanks-using-lightmeter">
        <!-- prettier-ignore -->
        <translate>Thank you for using Lightmeter.</translate>
        &copy;
        {{ year }} .
      </p>
      <p class="privacy-Policy">
        &bull;
        <a href="https://lightmeter.io/privacy-policy/" target="_blank">
          <!-- prettier-ignore -->
          <translate>Privacy Policy</translate>
        </a>
      </p>
      <p class="feedback">
        &bull;
        <a
          href=" mailto:hello@lightmeter.io?subject=Feedback%20on%20Lightmeter%20Control%20Center"
          target="_blank"
        >
          <!-- prettier-ignore -->
          <translate>Feedback</translate>
        </a>
        <!--onclick="_paq.push(['trackEvent', 'Feedback', 'clickMailTo']);"-->
      </p>
      <langauge-switcher></langauge-switcher>
    </div>
  </footer>
</template>
<script>
export default {
  name: "mainfooter",
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
#normal-footer {
  height: 32px;
}

#normal-footer .container {
  text-align: left;
}
#normal-footer .container p {
  margin-left: 0.5em;
  margin-right: 0.5em;
  margin-top: 0.8em;
}

#normal-footer .dropdown-menu li {
  font: normal normal normal 14px/20px Open Sans;
  color: #202324;
  text-align: left;
  padding-left: 14px;
  padding-right: 14px;
}

#normal-footer li:hover {
  text-decoration: none;
  background-color: #f8f9fa;
}

#normal-footer .dropdown {
  height: 28px;
  margin: 0rem !important;
  margin-left: 40px !important;
}

#normal-footer .dropdown button.dropdown-toggle,
#normal-footer .btn-secondary:not(:disabled):not(.disabled).active,
#normal-footer .btn-secondary:not(:disabled):not(.disabled):active,
#normal-footer .show > button.dropdown-toggle,
#normal-footer button.focus,
#normal-footer button:focus,
#normal-footer button:hover {
  background-color: #426475;
  border: 1px solid #426475;
  border-radius: 3px;
  text-align: left;
  font: normal normal normal 14px/20px Open Sans;
  letter-spacing: 0px;
  color: #ffffff;
  opacity: 1;
}

#normal-footer .container {
  margin-left: 0;
}
</style>
