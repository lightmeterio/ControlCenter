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
   >
    <b-carousel
      ref="walkthroughCarousel"
      v-model="slide"
      indicators
      img-width="860"
      img-height="580"
      :interval="interval"
    >
      <b-carousel-slide
        v-for="step of steps"
        v-bind:key="step.id"
        v-bind:img-src="step.picture"
      >
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
      <b-button variant="outline-dark" squared class="advance" @click="finish()" v-translate v-show="isLastStep">Finish</b-button>
    </div>

  </b-modal>
</template>

<script>

export default {
  props: {
    visible: Boolean
  },
  data() {
    return {
      slide: 0,
      interval: 0, // disables autoplay
      descriptions: [
        {
          "title": this.$gettext("All set up"),
          "description": this.$gettext("Postfix logs already imported. Blocks, blacklists and bounces are being analysed, among others."),
          "picture": "img/walkthrough/step1.png"
        },
        {
          "title": this.$gettext("Ready to use information"),
          "description": this.$gettext("Problems identified are displayed within Insights cards. Insights are generated for events happening while Control Center is running and also historical data."),
          "picture": "img/walkthrough/step2.png"
        },
        {
          "title": this.$gettext("Respond to problem reports faster"),
          "description": this.$gettext("A notification is triggered when any Local Insight is generated with high priority/secious status. Get the notifications to your mailbox or Slack channel."),
          "picture": "img/walkthrough/step3.png"
        },
        {
          "title": this.$gettext("Not just a peer, you are the network"),
          "description": this.$gettext("We build lightmeter to support a mission critical communication channel and help you unlock the power of Open Source infra."),
          "picture": "img/walkthrough/step4.png"
        },
        {
          "title": this.$gettext("Up to you"),
          "description": this.$gettext("Check the Settings page for setting up Slack and Email. Use a faulty IP to see lightmeter in action (e.g. 127.0.0.2). Use the feedback button to share your thoughts or get involved!"),
          "picture": "img/walkthrough/step5.png"
        }
      ]
    }
  },
  methods: {
    next() {
      this.$refs.walkthroughCarousel.next();
    },
    previous() {
      this.$refs.walkthroughCarousel.prev();
    },
    finish() {
      console.log("Finished!");
      this.$emit('finished');
    }
  },
  computed: {
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
  top: 25.5rem;
}

.walkthrough .carousel-indicators {
  top: 24rem;
}

@media (max-width: 991px) {
  .walkthrough .carousel-caption {
    top: 180px;
  }

  .walkthrough .carousel-indicators {
    display: none;
  }

  .walkthrough .carousel-item {
    height: 22rem;
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

</style>
