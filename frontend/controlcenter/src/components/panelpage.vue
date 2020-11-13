<template>
  <div class="panel-page">
    <b-container>
      <b-row class="justify-content-center vh-93">
        <b-col sm="5" class="align-self-center">
          <div class="panel panel-default mx-auto">
            <div class="panel-body">
              <b-row class="justify-content-center">
                <b-col lg="10">
                  <img
                    class="logo w-100"
                    src="@/assets/logo-color-120.png"
                    alt="Lightmeter logo"
                  />
                </b-col>
              </b-row>
              <slot></slot>
            </div>
          </div>
        </b-col>
      </b-row>
    </b-container>
    <auth-page-footer></auth-page-footer>
  </div>
</template>
<script>
export default {
  name: "panel-page"
};
</script>
<style></style>
