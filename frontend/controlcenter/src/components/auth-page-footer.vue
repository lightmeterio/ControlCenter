<template>
  <footer id="auth-page-footer" class="mt-auto">
    <b-container>
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
</style>
