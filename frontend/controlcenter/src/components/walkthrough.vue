<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <b-modal
    content-class="walkthrough"
    ref="walkthrough-modal"
    id="walkthrough-modal"
    size="lg"
    hide-footer
    centered
    :visible="visible"
    no-close-on-backdrop
    @hide="finish"
    @shown="onShown"
   >
    <b-carousel
      ref="walkthroughCarousel"
      v-model="currentSlide"
      indicators
      no-wrap
      :interval="interval"
    >
      <b-carousel-slide
        v-for="step of steps"
        v-bind:key="step.id"
        v-bind:img-src="step.picture"
      >
        <div slot="img">
          <div class="walkthrough-picture">
            <b-img :src="step.picture"></b-img>
          </div>
        </div>
        <div class="walkthrough-content">
          <div class="walkthrough-title">{{step.title}}</div>
          <div class="walkthrough-description">{{step.description}}</div>
        </div>
      </b-carousel-slide>
    </b-carousel>

    <div class="walkthrough-actions">
      <b-button variant="outline-dark" squared class="advance" @click="next()" v-translate v-show="isFirstStep">Let's Go!</b-button>
      <b-button variant="outline-dark" squared class="back" @click="previous()" v-translate v-show="!isFirstStep">
        <b-icon-arrow-left></b-icon-arrow-left>
        <span v-translate>Back</span>
      </b-button>
      <b-button variant="outline-dark" squared class="advance" @click="next()" v-translate v-show="!isFirstStep && !isLastStep">Continue</b-button>
      <b-button variant="outline-dark" squared class="advance" @click="hide()" v-translate v-show="isLastStep">Finish</b-button>
    </div>

  </b-modal>
</template>

<script>

import { getSettings } from "../lib/api.js";
import tracking from "../mixin/global_shared.js";

export default {
  mixins: [tracking],
  props: {
    visible: Boolean
  },
  data() {
    return {
      slide: 0,
      interval: 0, // disables autoplay
      descriptions: [
        {
          "title": this.$gettext("Nice Work!"),
          "description": this.$gettext("Your mailserver is now being monitored for blocks, blocklists, and bounces"),
          "picture": "img/walkthrough/step1.svg"
        },
        {
          "title": this.$gettext("Insights Incoming"),
          "description": this.$gettext("The homepage is the 'Observatory' – Insights will appear here, so watch this space!"),
          "picture": "img/walkthrough/step2.svg"
        },
        {
          "title": this.$gettext("Fastest Response"),
          "description": this.$gettext("High priority Insights trigger notifications – enable email or Slack for early warnings"),
          "picture": "img/walkthrough/step3.svg"
        },
        {
          "title": this.$gettext("You Are The Network"),
          "description": this.$gettext("Email is made up of peers like you – Lightmeter exists to support your independence"),
          "picture": "img/walkthrough/step4.svg"
        },
        {
          "title": this.$gettext("Over To You"),
          "description": this.$gettext("Check out the settings, add IPs to monitor, and enjoy your stay (restart this Walkthrough any time)"), 
          "picture": "img/walkthrough/step5.svg"
        }
      ]
    }
  },
  methods: {
    next() {
      this.trackEvent("WalkthroughNextStep", this.slide);
      this.$refs.walkthroughCarousel.next();
    },
    previous() {
      this.trackEvent("WalkthroughPrevStep", this.slide);
      this.$refs.walkthroughCarousel.prev();
    },
    onShown() {
      let vue = this;
      getSettings().then(function(response) {
        let evt = !response.data.walkthrough || !response.data.walkthrough.completed ?
          ["Walkthrough", "started"] :
          ["Walkthrough", "startedFooter"];
        vue.trackEvent(...evt);
      });
      this.currentSlide = 0;
    },
    hide () {
      this.$refs['walkthrough-modal'].hide();  // triggers finish()
    },
    finish() {
      let evt = this.isLastStep ? 
        ["Walkthrough", "finished"] :
        ["WalkthroughExited", this.slide];
      this.trackEvent(...evt);
      this.$emit('finished');
    },
  },
  computed: {
    currentSlide: {
      get() {
        return this.slide;
      },
      set(value) {
        this.slide = value;
      }
    },
    isFirstStep() {
      return this.slide == 0;
    },
    isLastStep() {
      return this.slide == this.descriptions.length - 1;
    },
    steps() {
      return this.descriptions.map(function(d, i) { return {"id": i, "title": d.title, "description": d.description, "picture": d.picture}; });
    }
  },
}

</script>

<style lang="less">

.walkthrough .carousel-item {
  height: 35rem;
}

.walkthrough .carousel-caption {
  top: 24rem;
}

.walkthrough .carousel-indicators {
  top: 23rem;
}

@media (max-width: 991px) {
  .walkthrough .carousel-caption {
    top: 18rem;
  }

  .walkthrough .carousel-indicators {
    display: none;
  }

  .walkthrough .carousel-item {
    height: 22rem;
  }

  .walkthrough .carousel-item {
    height: 72vh;
  }
}

.walkthrough .carousel-indicators li {
  width: 12px;
  height: 12px;
  background-color: #EFEFEF;
  border-radius: 6px;
  border: 0px;
}

.walkthrough .carousel-indicators li.active {
  background-color: #227AAF;
}

.walkthrough .walkthrough-description {
  margin-top: 1rem;
  color: black;
  font-size: 16px;
}

.walkthrough .walkthrough-title {
  color: black;
  font-size: 19px;
}

.walkthrough .modal-header {
  border-bottom: 0px;
}

.walkthrough .walkthrough-actions {
  margin-top: 1rem;
  margin-bottom: 1.5rem;
  display: flex;
  flex-direction: row;
  justify-content: center;
}

.walkthrough .walkthrough-actions > button {
  font-size: 14px;
  font-weight: bold;
  margin-left: 5px;
  margin-right: 5px;
  padding-left: 20px;
  padding-right: 20px;
  padding-top: 10px;
  padding-down: 10px;
}

.walkthrough .walkthrough-actions button.back {
  background-color: #ffffff;
  border: 0px;
}

.walkthrough .walkthrough-actions button.back span:last-child {
  margin-left: 10px;
}

.walkthrough .walkthrough-actions button.advance {
  background-color: #ffffff;
}

.walkthrough .walkthrough-actions button:hover {
  background-color: #ffffff;
  color: #000000;
}

.walkthrough .walkthrough-actions button.back:focus {
  border: 0px;
}

.walkthrough .walkthrough-content {
  margin: 0px;
  padding: 0px;
}

.walkthrough .walkthrough-picture {
  display: flex;
  justify-content: center;
}

</style>
