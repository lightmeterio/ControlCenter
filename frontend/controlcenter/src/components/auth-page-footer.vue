<template>
  <footer id="auth-page-footer" class="mt-auto">
    <b-container class="footer-text">
      Made with
      <svg
        class="bi bi-heart-fill"
        width="1em"
        height="1em"
        viewBox="0 0 16 16"
        fill="currentColor"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          fill-rule="evenodd"
          d="M8 1.314C12.438-3.248 23.534 4.735 8 15-7.534 4.736 3.562-3.248 8 1.314z"
        />
      </svg>
      by
      <a
        href="https://lightmeter.io"
        target="_blank"
        data-toggle="tooltip"
        data-placement="top"
        title="Lightmeter website"
        >Open Source professionals</a
      >
      <langauge-switcher></langauge-switcher>
    </b-container>
  </footer>
</template>

<script>
export default {
  name: "auth-page-footer"
};
</script>
<style>
#auth-page-footer {
  margin-bottom: 1em;
}
#auth-page-footer .container-right,
#auth-page-footer .container {
  display: flex;
  align-items: center;
  justify-content: center;
}

#auth-page-footer .dropdown li {
  font: normal normal normal 14px/20px Open Sans;
  color: #202324;
  text-align: left;
  padding-left: 14px;
  padding-right: 14px;
}

#auth-page-footer .dropdown {
  margin-left: 40px;
}

#auth-page-footer .dropdown button,
#auth-page-footer button:not(:disabled):not(.disabled).active,
#auth-page-footer button:not(:disabled):not(.disabled):active,
#auth-page-footer .show > button.dropdown-toggle,
#auth-page-footer button.focus,
#auth-page-footer button:focus,
#auth-page-footer button:hover {
  background-color: #f7f8f9;
  border: 1px solid #e6e7e7;
  border-radius: 3px;
  opacity: 1;
  text-align: left;
  font: normal normal normal 14px/20px Open Sans;
  letter-spacing: 0px;
  color: #202324;
  margin-left: 0.5em;
}

#auth-page-footer .dropdown-item {
  display: block;
  width: 100%;
  padding: 0.25rem 1.5rem;
  clear: both;
  white-space: nowrap;
  background-color: transparent;
  border: 0;
  font: normal normal normal 14px/20px Open Sans;
  color: #202324;
  text-align: left;
  padding-left: 14px;
  padding-right: 14px;
  text-decoration: none;
}

#auth-page-footer li:hover {
  text-decoration: none;
  background-color: #f8f9fa;
}

#auth-page-footer .footer-text svg,
#auth-page-footer .footer-text a {
  margin-left: 0.25em;
  margin-right: 0.25em;
}
</style>
