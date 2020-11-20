<template>
  <div id="app">
    <router-view />
  </div>
</template>

<style lang="less">
@font-face {
  font-family: Inter;
  src: url(~@/assets/fonts/inter/Inter-Regular.woff);
}

@font-face {
  font-family: Inter;
  src: url(~@/assets/fonts/inter/Inter-Italic.woff);
  font-style: italic;
}

@font-face {
  font-family: Inter;
  src: url(~@/assets/fonts/inter/Inter-Bold.woff);
  font-weight: bold;
}

@font-face {
  font-family: Inter;
  src: url(~@/assets/fonts/inter/Inter-BoldItalic.woff);
  font-weight: bold;
  font-style: italic;
}

/* todo(marcel) include missing fonts from open-sans  */

/* Extra widths, from https://github.com/twbs/bootstrap/issues/21943  */

/* Small devices (landscape phones, 576px and up)*/
@media (min-width: 576px) {
  .w-sm-100 {
    width: 100% !important;
  }
  .w-sm-75 {
    width: 75% !important;
  }
  .w-sm-50 {
    width: 50% !important;
  }
  .w-sm-25 {
    width: 25% !important;
  }
  .h-sm-100 {
    height: 100% !important;
  }
  .h-sm-75 {
    height: 75% !important;
  }
  .h-sm-50 {
    height: 50% !important;
  }
  .h-sm-25 {
    height: 25% !important;
  }
}

/* Medium devices (tablets, 768px and up)*/
@media (min-width: 768px) {
  .w-md-100 {
    width: 100% !important;
  }
  .w-md-75 {
    width: 75% !important;
  }
  .w-md-50 {
    width: 50% !important;
  }
  .w-md-25 {
    width: 25% !important;
  }
  .h-md-100 {
    height: 100% !important;
  }
  .h-md-75 {
    height: 75% !important;
  }
  .h-md-50 {
    height: 50% !important;
  }
  .h-md-25 {
    height: 25% !important;
  }
}

/* Large devices (desktops, 992px and up)*/
@media (min-width: 992px) {
  .w-lg-100 {
    width: 100% !important;
  }
  .w-lg-75 {
    width: 75% !important;
  }
  .w-lg-50 {
    width: 50% !important;
  }
  .w-lg-25 {
    width: 25% !important;
  }
  .h-lg-100 {
    height: 100% !important;
  }
  .h-lg-75 {
    height: 75% !important;
  }
  .h-lg-50 {
    height: 50% !important;
  }
  .h-lg-25 {
    height: 25% !important;
  }
}

/* Extra large devices (large desktops, 1200px and up)*/
@media (min-width: 1200px) {
  .w-xl-100 {
    width: 100% !important;
  }
  .w-xl-75 {
    width: 75% !important;
  }
  .w-xl-50 {
    width: 50% !important;
  }
  .w-xl-25 {
    width: 25% !important;
  }
  .h-xl-100 {
    height: 100% !important;
  }
  .h-xl-75 {
    height: 75% !important;
  }
  .h-xl-50 {
    height: 50% !important;
  }
  .h-xl-25 {
    height: 25% !important;
  }
}
/* end extra widths */

.align-right {
  text-align: right;
}

.align-left {
  text-align: left;
}

.align-center {
  text-align: center;
}

.vh-93 {
  min-height: 93vh;
}

.w-35 {
  width: 35%;
}

* {
  font-family: "Open Sans", sans-serif;
}

body {
  background: #fff;
}

h1,
h2,
h3,
h4,
h5,
h6 {
  font-family: Inter;
}

.main-content {
  margin-top: 0.6rem;
}

#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
}

footer {
  margin-top: 0.5rem;
}

footer,
footer a,
footer a:hover {
  background: #426475 0% 0% no-repeat padding-box;
  font: 12px/15px Open Sans;
  letter-spacing: 0px;
  color: #ffffff;
}

footer a {
  text-decoration: none;
}

footer p {
  margin: 0;
}

.input-group-addon {
  padding: 0 0.5rem;
  margin-bottom: 0;
  line-height: 1.25;
  text-align: center;
  background-color: #f7f8f9;
  border: 1px solid #ebebeb;
  border-left-style: solid;
  border-left-width: 1px;
  border-radius: 0.25rem;
  border-top-left-radius: 0.25rem;
  border-bottom-left-radius: 0.25rem;
}

.input-group-addon .svg-inline--fa {
  font-size: 1rem;
  color: #838685;
}

.input-group .form-control:not(:first-child),
.input-group-addon:not(:first-child),
.input-group-btn:not(:first-child) > .btn,
.input-group-btn:not(:first-child) > .btn-group > .btn,
.input-group-btn:not(:first-child) > .dropdown-toggle,
.input-group-btn:not(:last-child) > .btn-group:not(:first-child) > .btn,
.input-group-btn:not(:last-child) > .btn:not(:first-child) {
  border-top-left-radius: 0;
  border-bottom-left-radius: 0;
}

.input-group > .custom-select:not(:first-child),
.input-group > .form-control:not(:first-child) {
  border-top-left-radius: 0.25rem;
  border-bottom-left-radius: 0.25rem;
  border-right: 0;
}

.input-group-addon,
.input-group-btn {
  white-space: nowrap;
  vertical-align: middle;
}

.input-group .form-control,
.input-group-addon,
.input-group-btn {
  display: -ms-flexbox;
  display: flex;
  -ms-flex-align: center;
  align-items: center;
}

#show_hide_password {
  margin: 0.4rem 0;
}

#show_hide_password input {
  margin: 0;
}
</style>
