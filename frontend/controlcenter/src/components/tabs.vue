<template>
  <div
    :class="{
      'd-flex': variant === 'horizontal'
    }"
  >
    <ul
      :class="{
        'd-flex justify-content-between justify-content-lg-start text-left':
          variant === 'vertical'
      }"
    >
      <li
        v-for="(tab, index) in tabList"
        :key="index"
        :class="{
          '': index + 1 === activeTab,
          '': index + 1 !== activeTab
        }"
      >
        <label class="tabs" :for="`${_uid}${index}`" v-text="tab" />
        <input
          :id="`${_uid}${index}`"
          type="radio"
          :name="`${_uid}-tab`"
          :value="index + 1"
          v-model="activeTab"
          class="hidden"
        />
      </li>
    </ul>
    <template v-for="(tab, index) in tabList">
      <div :key="index" v-if="index + 1 === activeTab">
        <slot :name="`tabPanel-${index + 1}`" />
      </div>
    </template>
  </div>
</template>

<script>
export default {
  props: {
    tabList: {
      type: Array,
      required: true
    },
    variant: {
      type: String,
      required: false,
      default: () => "vertical",
      validator: value => ["horizontal", "vertical"].includes(value)
    }
  },
  data() {
    return {
      activeTab: 1
    };
  }
};
</script>

<style scoped>
.hidden {
  display: none !important;
}
ul {
  padding: 0;
  list-style: outside none none;
}
li {
  display: inline;
  padding: 1.5em 0;
}
label.tabs {
  padding: 0 2em 10px 2em;
  border-bottom: 3px solid #e5e7eb;
  cursor: pointer;
}
</style>
