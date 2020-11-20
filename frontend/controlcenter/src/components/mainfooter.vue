<template>
  <footer id="normal-footer" class="mt-auto">
    <div class="container d-flex">
      <p class="thanks-using-lightmeter">
        Thank you for using Lightmeter.
        <!--{{translate "Thank you for using Lightmeter."}}-->
        &copy;
        {{ year }} .
      </p>
      <p class="privacy-Policy">
        &bull;
        <a href="https://lightmeter.io/privacy-policy/" target="_blank"> </a>
        Privacy Policy
        <!--{{translate "Privacy Policy"}}-->
      </p>
      <p class="feedback">
        &bull;
        <a
          href=" mailto:hello@lightmeter.io?subject=Feedback%20on%20Lightmeter%20Control%20Center"
          target="_blank"
          >Feedback<!--{{translate "Feedback"}}--></a
        >
        <!--onclick="_paq.push(['trackEvent', 'Feedback', 'clickMailTo']);"-->
      </p>
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
#normal-footer .container {
  text-align: left;
  padding-top: 1em;
  padding-bottom: 0.5em;
}
#normal-footer .container p {
  margin-left: 0.5em;
  margin-right: 0.5em;
}
</style>
