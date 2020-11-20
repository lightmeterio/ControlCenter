<template>
  <div class="vue-daterange-picker" :class="{ inline: opens === 'inline' }">
    <div :class="controlContainerClass" @click="onClickPicker" ref="toggle">
      <!--
        Allows you to change the input which is visible before the picker opens

        @param {Date} startDate - current startDate
        @param {Date} endDate - current endDate
        @param {object} ranges - object with ranges
      -->
      <slot name="input" :startDate="start" :endDate="end" :ranges="ranges">
        <i class="glyphicon glyphicon-calendar fa fa-calendar"></i>&nbsp;
        <span>{{ rangeText }}</span>
        <b class="caret"></b>
      </slot>
    </div>
    <transition name="slide-fade" mode="out-in">
      <div
        class="daterangepicker dropdown-menu ltr"
        :class="pickerStyles"
        v-if="open || opens === 'inline'"
        v-append-to-body
        ref="dropdown"
      >
        <!--
          Optional header slot (same props as footer) @see footer slot for documentation
        -->
        <slot
          name="header"
          :rangeText="rangeText"
          :locale="locale"
          :clickCancel="clickCancel"
          :clickApply="clickedApply"
          :in_selection="in_selection"
          :autoApply="autoApply"
        >
        </slot>

        <div class="calendars row no-gutters">
          <!--
            Allows you to change the range

            @param {Date} startDate - current startDate
            @param {Date} endDate - current endDate
            @param {object} ranges - object with ranges
            @param {Fn} clickRange(dateRange) - call to select the dateRange - any two date objects or an object from tha ranges array
          -->
          <slot
            name="ranges"
            :startDate="start"
            :endDate="end"
            :ranges="ranges"
            :clickRange="clickRange"
            v-if="showRanges"
          >
            <calendar-ranges
              class="col-12 col-md-auto"
              @clickRange="clickRange"
              @showCustomRange="showCustomRangeCalendars = true"
              :always-show-calendars="alwaysShowCalendars"
              :locale-data="locale"
              :ranges="ranges"
              :selected="{ startDate: start, endDate: end }"
            ></calendar-ranges>
          </slot>

          <div class="calendars-container" v-if="showCalendars">
            <div
              class="drp-calendar col left"
              :class="{ single: singleDatePicker }"
            >
              <div class="daterangepicker_input d-none d-sm-block" v-if="false">
                <input
                  class="input-mini form-control"
                  type="text"
                  name="daterangepicker_start"
                  :value="startText"
                />
                <i class="fa fa-calendar glyphicon glyphicon-calendar"></i>
              </div>
              <div class="calendar-table">
                <calendar
                  :monthDate="monthDate"
                  :locale-data="locale"
                  :start="start"
                  :end="end"
                  :minDate="min"
                  :maxDate="max"
                  :show-dropdowns="showDropdowns"
                  @change-month="changeLeftMonth"
                  :date-format="dateFormatFn"
                  @dateClick="dateClick"
                  @hoverDate="hoverDate"
                  :showWeekNumbers="showWeekNumbers"
                ></calendar>
              </div>
              <calendar-time
                v-if="timePicker && start"
                @update="onUpdateStartTime"
                :miniute-increment="timePickerIncrement"
                :hour24="timePicker24Hour"
                :second-picker="timePickerSeconds"
                :current-time="start"
                :readonly="readonly"
              />
            </div>

            <div class="drp-calendar col right" v-if="!singleDatePicker">
              <div class="daterangepicker_input" v-if="false">
                <input
                  class="input-mini form-control"
                  type="text"
                  name="daterangepicker_end"
                  :value="endText"
                />
                <i class="fa fa-calendar glyphicon glyphicon-calendar"></i>
              </div>
              <div class="calendar-table">
                <calendar
                  :monthDate="nextMonthDate"
                  :locale-data="locale"
                  :start="start"
                  :end="end"
                  :minDate="min"
                  :maxDate="max"
                  :show-dropdowns="showDropdowns"
                  @change-month="changeRightMonth"
                  :date-format="dateFormatFn"
                  @dateClick="dateClick"
                  @hoverDate="hoverDate"
                  :showWeekNumbers="showWeekNumbers"
                ></calendar>
              </div>
              <calendar-time
                v-if="timePicker && end"
                @update="onUpdateEndTime"
                :miniute-increment="timePickerIncrement"
                :hour24="timePicker24Hour"
                :second-picker="timePickerSeconds"
                :current-time="end"
                :readonly="readonly"
              />
            </div>
          </div>
        </div>
        <!--
          Allows you to change footer of the component (where the buttons are)

          @param {string} rangeText - the formatted date range by the component
          @param {object} locale - the locale object @see locale prop
          @param {function} clickCancel - function which is called when you want to cancel the range picking and reset old values
          @param {function} clickApply -function which to call when you want to apply the selection
          @param {boolean} in_selection - is the picker in selection mode
          @param {boolean} autoApply - value of the autoApply prop (whether to select immediately)
        -->
        <slot
          name="footer"
          :rangeText="rangeText"
          :locale="locale"
          :clickCancel="clickCancel"
          :clickApply="clickedApply"
          :in_selection="in_selection"
          :autoApply="autoApply"
        >
          <div class="drp-buttons" v-if="!autoApply">
            <span class="drp-selected" v-if="showCalendars">{{
              rangeText
            }}</span>
            <button
              class="cancelBtn btn btn-sm btn-secondary"
              type="button"
              @click="clickCancel"
              v-if="!readonly"
            >
              {{ locale.cancelLabel }}
            </button>
            <button
              class="applyBtn btn btn-sm btn-success"
              :disabled="in_selection"
              type="button"
              @click="clickedApply"
              v-if="!readonly"
            >
              {{ locale.applyLabel }}
            </button>
          </div>
        </slot>
      </div>
    </transition>
  </div>
</template>

<style>
.daterangepicker.dropdown-menu {
  display: block;
}
td[data-v-aab6e828],
th[data-v-aab6e828] {
  padding: 2px;
  background-color: #fff;
}
td.today[data-v-aab6e828] {
  font-weight: 700;
}
td.disabled[data-v-aab6e828] {
  pointer-events: none;
  background-color: #eee;
  border-radius: 0;
  opacity: 0.6;
}
.fa[data-v-aab6e828] {
  display: inline-block;
  width: 100%;
  height: 100%;
  background: transparent no-repeat 50%;
  background-size: 100% 100%;
  fill: #ccc;
}
.next[data-v-aab6e828]:hover,
.prev[data-v-aab6e828]:hover {
  background-color: transparent !important;
}
.next .fa[data-v-aab6e828]:hover,
.prev .fa[data-v-aab6e828]:hover {
  opacity: 0.6;
}
.chevron-left[data-v-aab6e828] {
  width: 16px;
  height: 16px;
  display: block;
  background-image: url("data:image/svg+xml;charset=utf8,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='-2 -2 10 10'%3E%3Cpath d='M5.25 0l-4 4 4 4 1.5-1.5-2.5-2.5 2.5-2.5-1.5-1.5z'/%3E%3C/svg%3E");
}
.chevron-right[data-v-aab6e828] {
  width: 16px;
  height: 16px;
  display: block;
  background-image: url("data:image/svg+xml;charset=utf8,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='-2 -2 10 10'%3E%3Cpath d='M2.75 0l-1.5 1.5 2.5 2.5-2.5 2.5 1.5 1.5 4-4-4-4z'/%3E%3C/svg%3E");
}
.yearselect[data-v-aab6e828] {
  padding-right: 1px;
  border: none;
  -webkit-appearance: menulist;
  -moz-appearance: menulist;
  appearance: menulist;
}
.monthselect[data-v-aab6e828] {
  border: none;
}
.drp-calendar .col .left {
  -webkit-box-flex: 0;
  -ms-flex: 0 0 auto;
  flex: 0 0 auto;
}
.daterangepicker.hide-calendars.show-ranges .ranges,
.daterangepicker.hide-calendars.show-ranges .ranges ul {
  width: 100%;
}
.daterangepicker .calendars-container {
  display: -webkit-box;
  display: -ms-flexbox;
  display: flex;
}
.daterangepicker[readonly] {
  pointer-events: none;
}
.daterangepicker {
  position: absolute;
  color: inherit;
  background-color: #fff;
  border-radius: 4px;
  border: 1px solid #ddd;
  width: 278px;
  max-width: none;
  padding: 0;
  margin-top: 7px;
  top: 100px;
  left: 20px;
  z-index: 3001;
  display: none;
  font-size: 15px;
  line-height: 1em;
}
.daterangepicker:after,
.daterangepicker:before {
  position: absolute;
  display: inline-block;
  border-bottom-color: rgba(0, 0, 0, 0.2);
  content: "";
}
.daterangepicker:before {
  top: -7px;
  border-right: 7px solid transparent;
  border-left: 7px solid transparent;
  border-bottom: 7px solid #ccc;
}
.daterangepicker:after {
  top: -6px;
  border-right: 6px solid transparent;
  border-bottom: 6px solid #fff;
  border-left: 6px solid transparent;
}
.daterangepicker.opensleft:before {
  right: 9px;
}
.daterangepicker.opensleft:after {
  right: 10px;
}
.daterangepicker.openscenter:after,
.daterangepicker.openscenter:before {
  left: 0;
  right: 0;
  width: 0;
  margin-left: auto;
  margin-right: auto;
}
.daterangepicker.opensright:before {
  left: 9px;
}
.daterangepicker.opensright:after {
  left: 10px;
}
.daterangepicker.drop-up {
  margin-top: -7px;
}
.daterangepicker.drop-up:before {
  top: auto;
  bottom: -7px;
  border-bottom: initial;
  border-top: 7px solid #ccc;
}
.daterangepicker.drop-up:after {
  top: auto;
  bottom: -6px;
  border-bottom: initial;
  border-top: 6px solid #fff;
}
.daterangepicker.single .drp-selected {
  display: none;
}
.daterangepicker.show-calendar .drp-buttons,
.daterangepicker.show-calendar .drp-calendar {
  display: block;
}
.daterangepicker.auto-apply .drp-buttons {
  display: none;
}
.daterangepicker .drp-calendar {
  display: none;
  max-width: 270px;
  width: 270px;
}
.daterangepicker .drp-calendar.left {
  padding: 8px 0 8px 8px;
}
.daterangepicker .drp-calendar.right {
  padding: 8px;
}
.daterangepicker .drp-calendar.single .calendar-table {
  border: none;
}
.daterangepicker .calendar-table .next span,
.daterangepicker .calendar-table .prev span {
  color: #fff;
  border: solid #000;
  border-width: 0 2px 2px 0;
  border-radius: 0;
  display: inline-block;
  padding: 3px;
}
.daterangepicker .calendar-table .next span {
  transform: rotate(-45deg);
  -webkit-transform: rotate(-45deg);
}
.daterangepicker .calendar-table .prev span {
  transform: rotate(135deg);
  -webkit-transform: rotate(135deg);
}
.daterangepicker .calendar-table td,
.daterangepicker .calendar-table th {
  white-space: nowrap;
  text-align: center;
  vertical-align: middle;
  min-width: 32px;
  width: 32px;
  height: 24px;
  line-height: 24px;
  font-size: 12px;
  border-radius: 4px;
  border: 1px solid transparent;
  cursor: pointer;
}
.daterangepicker .calendar-table {
  border: 1px solid #fff;
  border-radius: 4px;
  background-color: #fff;
}
.daterangepicker .calendar-table table {
  width: 100%;
  margin: 0;
  border-spacing: 0;
  border-collapse: collapse;
  display: table;
}
.daterangepicker td.available:hover,
.daterangepicker th.available:hover {
  background-color: #eee;
  border-color: transparent;
  color: inherit;
}
.daterangepicker td.week,
.daterangepicker th.week {
  font-size: 80%;
  color: #ccc;
}
.daterangepicker td.off,
.daterangepicker td.off.end-date,
.daterangepicker td.off.in-range,
.daterangepicker td.off.start-date {
  background-color: #fff;
  border-color: transparent;
  color: #999;
}
.daterangepicker td.in-range {
  background-color: #ebf4f8;
  border-color: transparent;
  color: #000;
  border-radius: 0;
}
.daterangepicker td.start-date {
  border-radius: 4px 0 0 4px;
}
.daterangepicker td.end-date {
  border-radius: 0 4px 4px 0;
}
.daterangepicker td.start-date.end-date {
  border-radius: 4px;
}
.daterangepicker td.active,
.daterangepicker td.active:hover {
  background-color: #357ebd;
  border-color: transparent;
  color: #fff;
}
.daterangepicker th.month {
  width: auto;
}
.daterangepicker option.disabled,
.daterangepicker td.disabled {
  color: #999;
  cursor: not-allowed;
  text-decoration: line-through;
}
.daterangepicker select.monthselect,
.daterangepicker select.yearselect {
  font-size: 12px;
  padding: 1px;
  height: auto;
  margin: 0;
  cursor: default;
}
.daterangepicker select.monthselect {
  margin-right: 2%;
  width: 56%;
}
.daterangepicker select.yearselect {
  width: 40%;
}
.daterangepicker select.ampmselect,
.daterangepicker select.hourselect,
.daterangepicker select.minuteselect,
.daterangepicker select.secondselect {
  width: 50px;
  margin: 0 auto;
  background: #eee;
  border: 1px solid #eee;
  padding: 2px;
  outline: 0;
  font-size: 12px;
}
.daterangepicker .calendar-time {
  text-align: center;
  margin: 4px auto 0 auto;
  line-height: 30px;
  position: relative;
  display: -webkit-box;
  display: -ms-flexbox;
  display: flex;
}
.daterangepicker .calendar-time select.disabled {
  color: #ccc;
  cursor: not-allowed;
}
.daterangepicker .drp-buttons {
  clear: both;
  text-align: right;
  padding: 8px;
  border-top: 1px solid #ddd;
  display: none;
  line-height: 12px;
  vertical-align: middle;
}
.daterangepicker .drp-selected {
  display: inline-block;
  font-size: 12px;
  padding-right: 8px;
}
.daterangepicker .drp-buttons .btn {
  margin-left: 8px;
  font-size: 12px;
  font-weight: 700;
  padding: 4px 8px;
}
.daterangepicker.show-ranges .drp-calendar.left {
  border-left: 1px solid #ddd;
}
.daterangepicker .ranges {
  text-align: left;
  margin: 0;
}
.daterangepicker.show-calendar .ranges {
  margin-top: 8px;
}
.daterangepicker .ranges ul {
  list-style: none;
  margin: 0 auto;
  padding: 0;
  width: 100%;
}
.daterangepicker .ranges li {
  font-size: 12px;
  padding: 8px 12px;
  cursor: pointer;
}
.daterangepicker .ranges li:hover {
  background-color: #eee;
  color: #000;
}
.daterangepicker .ranges li.active {
  background-color: #08c;
  color: #fff;
}
@media (min-width: 564px) {
  .daterangepicker {
    width: auto;
  }
  .daterangepicker .ranges ul {
    width: 140px;
  }
  .daterangepicker.single .ranges ul {
    width: 100%;
  }
  .daterangepicker.single .drp-calendar.left {
    clear: none;
  }
  .daterangepicker.ltr {
    direction: ltr;
    text-align: left;
  }
  .daterangepicker.ltr .drp-calendar.left {
    clear: left;
    margin-right: 0;
  }
  .daterangepicker.ltr .drp-calendar.left .calendar-table {
    border-right: none;
    border-top-right-radius: 0;
    border-bottom-right-radius: 0;
  }
  .daterangepicker.ltr .drp-calendar.right {
    margin-left: 0;
  }
  .daterangepicker.ltr .drp-calendar.right .calendar-table {
    border-left: none;
    border-top-left-radius: 0;
    border-bottom-left-radius: 0;
  }
  .daterangepicker.ltr .drp-calendar.left .calendar-table {
    padding-right: 8px;
  }
  .daterangepicker.rtl {
    direction: rtl;
    text-align: right;
  }
  .daterangepicker.rtl .drp-calendar.left {
    clear: right;
    margin-left: 0;
  }
  .daterangepicker.rtl .drp-calendar.left .calendar-table {
    border-left: none;
    border-top-left-radius: 0;
    border-bottom-left-radius: 0;
  }
  .daterangepicker.rtl .drp-calendar.right {
    margin-right: 0;
  }
  .daterangepicker.rtl .drp-calendar.right .calendar-table {
    border-right: none;
    border-top-right-radius: 0;
    border-bottom-right-radius: 0;
  }
  .daterangepicker.rtl .drp-calendar.left .calendar-table {
    padding-left: 12px;
  }
  .daterangepicker.rtl .drp-calendar,
  .daterangepicker.rtl .ranges {
    text-align: right;
  }
}
@media (min-width: 730px) {
  .daterangepicker .ranges {
    width: auto;
  }
  .daterangepicker .drp-calendar.left {
    clear: none !important;
  }
}
.reportrange-text[data-v-4f8eb193] {
  background: #fff;
  cursor: pointer;
  padding: 5px 10px;
  border: 1px solid #ccc;
  width: 100%;
}
.daterangepicker[data-v-4f8eb193] {
  -webkit-box-orient: vertical;
  -webkit-box-direction: normal;
  -ms-flex-direction: column;
  flex-direction: column;
  display: -webkit-box;
  display: -ms-flexbox;
  display: flex;
  width: auto;
}
@media screen and (max-width: 768px) {
  .daterangepicker.show-ranges .drp-calendar.left[data-v-4f8eb193] {
    border-left: 0;
  }
  .daterangepicker.show-ranges .ranges[data-v-4f8eb193] {
    border-bottom: 1px solid #ddd;
  }
  .daterangepicker.show-ranges .ranges[data-v-4f8eb193] ul {
    display: -webkit-box;
    display: -ms-flexbox;
    display: flex;
    -ms-flex-wrap: wrap;
    flex-wrap: wrap;
    width: auto;
  }
}
@media screen and (max-width: 541px) {
  .daterangepicker .calendars-container[data-v-4f8eb193] {
    -ms-flex-wrap: wrap;
    flex-wrap: wrap;
  }
}
@media screen and (min-width: 540px) {
  .daterangepicker.show-weeknumbers[data-v-4f8eb193],
  .daterangepicker[data-v-4f8eb193] {
    min-width: 486px;
  }
}
@media screen and (min-width: 768px) {
  .daterangepicker.show-ranges.show-weeknumbers[data-v-4f8eb193],
  .daterangepicker.show-ranges[data-v-4f8eb193] {
    min-width: 682px;
  }
}
@media screen and (max-width: 340px) {
  .daterangepicker.single.show-weeknumbers[data-v-4f8eb193],
  .daterangepicker.single[data-v-4f8eb193] {
    min-width: 250px;
  }
}
@media screen and (min-width: 339px) {
  .daterangepicker.single[data-v-4f8eb193] {
    min-width: auto;
  }
  .daterangepicker.single.show-ranges.show-weeknumbers[data-v-4f8eb193],
  .daterangepicker.single.show-ranges[data-v-4f8eb193] {
    min-width: 356px;
  }
  .daterangepicker.single.show-ranges .drp-calendar.left[data-v-4f8eb193] {
    border-left: 1px solid #ddd;
  }
  .daterangepicker.single.show-ranges .ranges[data-v-4f8eb193] {
    width: auto;
    max-width: none;
    -ms-flex-preferred-size: auto;
    flex-basis: auto;
    border-bottom: 0;
  }
  .daterangepicker.single.show-ranges .ranges[data-v-4f8eb193] ul {
    display: block;
    width: 100%;
  }
}
.daterangepicker.show-calendar[data-v-4f8eb193] {
  display: block;
  top: auto;
}
.daterangepicker.opensleft[data-v-4f8eb193] {
  right: 10px;
  left: auto;
}
.daterangepicker.openscenter[data-v-4f8eb193] {
  right: auto;
  left: 50%;
  -webkit-transform: translate(-50%);
  transform: translate(-50%);
}
.daterangepicker.opensright[data-v-4f8eb193] {
  left: 10px;
  right: auto;
}
.slide-fade-enter-active[data-v-4f8eb193] {
  -webkit-transition: all 0.2s ease;
  transition: all 0.2s ease;
}
.slide-fade-leave-active[data-v-4f8eb193] {
  -webkit-transition: all 0.1s cubic-bezier(1, 0.5, 0.8, 1);
  transition: all 0.1s cubic-bezier(1, 0.5, 0.8, 1);
}
.slide-fade-enter[data-v-4f8eb193],
.slide-fade-leave-to[data-v-4f8eb193] {
  -webkit-transform: translateX(10px);
  transform: translateX(10px);
  opacity: 0;
}
.vue-daterange-picker[data-v-4f8eb193] {
  position: relative;
  display: inline-block;
  min-width: 60px;
}
.vue-daterange-picker .dropdown-menu[data-v-4f8eb193] {
  padding: 0;
}
.vue-daterange-picker .show-ranges.hide-calendars[data-v-4f8eb193] {
  width: 150px;
  min-width: 150px;
}
.inline .daterangepicker[data-v-4f8eb193] {
  position: static;
}
.inline .daterangepicker[data-v-4f8eb193]:after,
.inline .daterangepicker[data-v-4f8eb193]:before {
  display: none;
}
</style>

<script>
import dateUtilMixin from "./dateUtilMixin.js";
import Calendar from "./Calendar.vue";
import CalendarTime from "./CalendarTime.vue";
import CalendarRanges from "./CalendarRanges.vue";
import { getDateUtil } from "./util.js";
import appendToBody from "./directives/appendToBody.js";

export default {
  inheritAttrs: false,
  components: { Calendar, CalendarTime, CalendarRanges },
  mixins: [dateUtilMixin],
  directives: { appendToBody },
  model: {
    prop: "dateRange",
    event: "update"
  },
  props: {
    /**
     * minimum date allowed to be selected
     * @default null
     */
    minDate: {
      type: [String, Date],
      default() {
        return null;
      }
    },
    /**
     * maximum date allowed to be selected
     * @default null
     */
    maxDate: {
      type: [String, Date],
      default() {
        return null;
      }
    },
    /**
     * Show the week numbers on the left side of the calendar
     */
    showWeekNumbers: {
      type: Boolean,
      default: false
    },
    /**
     * Each calendar has separate navigation when this is false
     */
    linkedCalendars: {
      type: Boolean,
      default: true
    },
    /**
     * Only show a single calendar, with or without ranges.
     *
     * Set true or 'single' for a single calendar with no ranges, single dates only.
     * Set 'range' for a single calendar WITH ranges.
     * Set false for a double calendar with ranges.
     */
    singleDatePicker: {
      type: [Boolean, String],
      default: false
    },
    /**
     * Show the dropdowns for month and year selection above the calendars
     */
    showDropdowns: {
      type: Boolean,
      default: false
    },
    /**
     * Show the dropdowns for time (hour/minute) selection below the calendars
     */
    timePicker: {
      type: Boolean,
      default: false
    },
    /**
     * Determines the increment of minutes in the minute dropdown
     */
    timePickerIncrement: {
      type: Number,
      default: 5
    },
    /**
     * Use 24h format for the time
     */
    timePicker24Hour: {
      type: Boolean,
      default: true
    },
    /**
     * Allows you to select seconds except hour/minute
     */
    timePickerSeconds: {
      type: Boolean,
      default: false
    },
    /**
     * Auto apply selected range. If false you need to click an apply button
     */
    autoApply: {
      type: Boolean,
      default: false
    },
    /**
     * Object containing locale data used by the picker. See example below the table
     *
     * @default *see below
     */
    localeData: {
      type: Object,
      default() {
        return {};
      }
    },
    /**
     * This is the v-model prop which the component uses. This should be an object containing startDate and endDate props.
     * Each of the props should be a string which can be parsed by Date, or preferably a Date Object.
     * @default {
     * startDate: null,
     * endDate: null
     * }
     */
    dateRange: {
      // for v-model
      type: [Object],
      default: null,
      required: true
    },
    /**
     * You can set this to false in order to hide the ranges selection. Otherwise it is an object with key/value. See below
     * @default *see below
     */
    ranges: {
      type: [Object, Boolean],
      default() {
        let today = new Date();
        today.setHours(0, 0, 0, 0);

        let yesterday = new Date();
        yesterday.setDate(today.getDate() - 1);
        yesterday.setHours(0, 0, 0, 0);

        let thisMonthStart = new Date(today.getFullYear(), today.getMonth(), 1);
        let thisMonthEnd = new Date(
          today.getFullYear(),
          today.getMonth() + 1,
          0
        );

        return {
          Today: [today, today],
          Yesterday: [yesterday, yesterday],
          "This month": [thisMonthStart, thisMonthEnd],
          "This year": [
            new Date(today.getFullYear(), 0, 1),
            new Date(today.getFullYear(), 11, 31)
          ],
          "Last month": [
            new Date(today.getFullYear(), today.getMonth() - 1, 1),
            new Date(today.getFullYear(), today.getMonth(), 0)
          ]
        };
      }
    },
    /**
     * which way the picker opens - "center", "left", "right" or "inline"
     */
    opens: {
      type: String,
      default: "center"
    },
    /**
       function(classes, date) - special prop type function which accepts 2 params:
       "classes" - the classes that the component's logic has defined,
       "date" - tha date currently processed.
       You should return Vue class object which should be applied to the date rendered.
       */
    dateFormat: Function,
    /**
     * If set to false and one of the predefined ranges is selected then calendars are hidden.
     * If no range is selected or you have clicked the "Custom ranges" then the calendars are shown.
     */
    alwaysShowCalendars: {
      type: Boolean,
      default: true
    },
    /**
     * Disabled state. If true picker do not popup on click.
     */
    disabled: {
      type: Boolean,
      default: false
    },
    /**
     * Class of html picker control container
     */
    controlContainerClass: {
      type: [Object, String],
      default: "form-control reportrange-text"
    },
    /**
     * Append the dropdown element to the end of the body
     * and size/position it dynamically. Use it if you have
     * overflow or z-index issues.
     * @type {Boolean}
     */
    appendToBody: {
      type: Boolean,
      default: false
    },
    /**
     * When `appendToBody` is true, this function is responsible for
     * positioning the drop down list.
     *
     * If a function is returned from `calculatePosition`, it will
     * be called when the drop down list is removed from the DOM.
     * This allows for any garbage collection you may need to do.
     *
     * @since v0.5.1
     */
    calculatePosition: {
      type: Function,
      /**
       * @param dropdownList {HTMLUListElement}
       * @param component {Vue} current instance of vue date range picker
       * @param width {int} calculated width in pixels of the dropdown menu
       * @param top {int} absolute position top value in pixels relative to the document
       * @param left {int} absolute position left value in pixels relative to the document
       * @param right {int} absolute position right value in pixels relative to the document
       * @return {function|void}
       */
      default(dropdownList, component, { width, top, left, right }) {
        // which way the picker opens - "center", "left" or "right"
        if (component.opens === "center") {
          // console.log('center open', left, width)
          dropdownList.style.left = left + width / 2 + "px";
        } else if (component.opens === "left") {
          // console.log('left open', right, width)
          dropdownList.style.right = window.innerWidth - right + "px";
        } else if (component.opens === "right") {
          // console.log('right open')
          dropdownList.style.left = left + "px";
        }
        dropdownList.style.top = top + "px";
        // dropdownList.style.width = width + 'px'
      }
    },
    /**
     * Whether to close the dropdown on "esc"
     */
    closeOnEsc: {
      type: Boolean,
      default: true
    },
    /**
     * Makes the picker readonly. No button in footer. No ranges. Cannot change.
     */
    readonly: {
      type: Boolean
    }
  },
  data() {
    //copy locale data object
    const util = getDateUtil();
    let data = { locale: util.localeData({ ...this.localeData }) };

    let startDate = this.dateRange.startDate || null;
    let endDate = this.dateRange.endDate || null;

    data.monthDate = startDate ? new Date(startDate) : new Date();
    //get next month date
    data.nextMonthDate = util.nextMonth(data.monthDate);

    data.start = startDate ? new Date(startDate) : null;
    if (this.singleDatePicker && this.singleDatePicker !== "range") {
      // ignore endDate for singleDatePicker
      data.end = data.start;
    } else {
      data.end = endDate ? new Date(endDate) : null;
    }
    data.in_selection = false;
    data.open = false;
    //When alwaysShowCalendars = false and custom range is clicked
    data.showCustomRangeCalendars = false;

    // update day names order to firstDay
    if (data.locale.firstDay !== 0) {
      let iterator = data.locale.firstDay;
      let weekDays = [...data.locale.daysOfWeek];
      while (iterator > 0) {
        weekDays.push(weekDays.shift());
        iterator--;
      }
      data.locale.daysOfWeek = weekDays;
    }
    return data;
  },
  methods: {
    dateFormatFn(classes, date) {
      let dt = new Date(date);
      dt.setHours(0, 0, 0, 0);
      let start = new Date(this.start);
      start.setHours(0, 0, 0, 0);
      let end = new Date(this.end);
      end.setHours(0, 0, 0, 0);

      classes["in-range"] = dt >= start && dt <= end;

      return this.dateFormat ? this.dateFormat(classes, date) : classes;
    },
    changeLeftMonth(value) {
      let newDate = new Date(value.year, value.month - 1, 1);
      this.monthDate = newDate;
      if (
        this.linkedCalendars ||
        this.$dateUtil.yearMonth(this.monthDate) >=
          this.$dateUtil.yearMonth(this.nextMonthDate)
      ) {
        this.nextMonthDate = this.$dateUtil.validateDateRange(
          this.$dateUtil.nextMonth(newDate),
          this.minDate,
          this.maxDate
        );
        if (
          (!this.singleDatePicker || this.singleDatePicker === "range") &&
          this.$dateUtil.yearMonth(this.monthDate) ===
            this.$dateUtil.yearMonth(this.nextMonthDate)
        ) {
          this.monthDate = this.$dateUtil.validateDateRange(
            this.$dateUtil.prevMonth(this.monthDate),
            this.minDate,
            this.maxDate
          );
        }
      }
      /**
       * Emits event when the viewing month is changes. The second param is the index of the calendar.
       *
       * @param {monthDate} date displayed (first day of the month)
       * @param calendarIndex int 0 - first(left) calendar, 1 - second(right) calendar
       */
      this.$emit("change-month", this.monthDate, 0);
    },
    changeRightMonth(value) {
      let newDate = new Date(value.year, value.month - 1, 1);
      this.nextMonthDate = newDate;
      if (
        this.linkedCalendars ||
        this.$dateUtil.yearMonth(this.nextMonthDate) <=
          this.$dateUtil.yearMonth(this.monthDate)
      ) {
        this.monthDate = this.$dateUtil.validateDateRange(
          this.$dateUtil.prevMonth(newDate),
          this.minDate,
          this.maxDate
        );
        if (
          this.$dateUtil.yearMonth(this.monthDate) ===
          this.$dateUtil.yearMonth(this.nextMonthDate)
        ) {
          this.nextMonthDate = this.$dateUtil.validateDateRange(
            this.$dateUtil.nextMonth(this.nextMonthDate),
            this.minDate,
            this.maxDate
          );
        }
      }
      this.$emit("change-month", this.monthDate, 1);
    },
    normalizeDatetime(value, oldValue) {
      let newDate = new Date(value);
      if (this.timePicker && oldValue) {
        newDate.setHours(oldValue.getHours());
        newDate.setMinutes(oldValue.getMinutes());
        newDate.setSeconds(oldValue.getSeconds());
        newDate.setMilliseconds(oldValue.getMilliseconds());
      }

      return newDate;
    },
    dateClick(value) {
      if (this.readonly) return false;
      if (this.in_selection) {
        this.in_selection = false;
        this.end = this.normalizeDatetime(value, this.end);

        if (this.end < this.start) {
          this.in_selection = true;
          this.start = this.normalizeDatetime(value, this.start);
        }
        if (!this.in_selection) {
          this.onSelect();
          if (this.autoApply) this.clickedApply();
        }
      } else {
        this.start = this.normalizeDatetime(value, this.start);
        this.end = this.normalizeDatetime(value, this.end);
        if (!this.singleDatePicker || this.singleDatePicker === "range") {
          this.in_selection = true;
        } else {
          this.onSelect();
          if (this.autoApply) this.clickedApply();
        }
      }
    },
    hoverDate(value) {
      if (this.readonly) return false;
      let dt = this.normalizeDatetime(value, this.end);
      if (this.in_selection && dt >= this.start) this.end = dt;
      /**
       * Emits event when the mouse hovers a date
       * @param {Date} value the date that is being hovered
       */
      this.$emit("hoverDate", value);
    },
    onClickPicker() {
      if (!this.disabled) {
        this.togglePicker(null, true);
      }
    },
    togglePicker(value, event) {
      if (typeof value === "boolean") {
        this.open = value;
      } else {
        this.open = !this.open;
      }

      if (event === true)
        /**
         * Emits whenever the picker opens/closes
         * @param {boolean} open - the current state of the picker
         * @param {Function} togglePicker - function (show, event) which can be used to control the picker. where "show" is the new state and "event" is boolean indicating whether a new event should be raised
         */
        this.$emit("toggle", this.open, this.togglePicker);
    },
    clickedApply() {
      // this.open = false
      this.togglePicker(false, true);
      /**
       * Emits when the user selects a range from the picker and clicks "apply" (if autoApply is true).
       * @param {json} value - json object containing the dates: {startDate, endDate}
       */
      this.$emit("update", {
        startDate: this.start,
        endDate:
          this.singleDatePicker && this.singleDatePicker !== "range"
            ? this.start
            : this.end
      });
    },
    clickCancel() {
      if (this.open) {
        // reset start and end
        let startDate = this.dateRange.startDate;
        let endDate = this.dateRange.endDate;
        this.start = startDate ? new Date(startDate) : null;
        this.end = endDate ? new Date(endDate) : null;
        // this.open = false
        this.togglePicker(false, true);
      }
    },
    onSelect() {
      /**
       * Emits when the user selects a range from the picker.
       * @param {json} value - json object containing the dates: {startDate, endDate}
       */
      this.$emit("select", { startDate: this.start, endDate: this.end });
    },
    clickAway($event) {
      if (
        $event &&
        $event.target &&
        !this.$el.contains($event.target) &&
        this.$refs.dropdown &&
        !this.$refs.dropdown.contains($event.target)
      ) {
        this.clickCancel();
      }
    },
    clickRange(value) {
      this.in_selection = false;

      if (
        this.$dateUtil.isValidDate(value[0]) &&
        this.$dateUtil.isValidDate(value[1])
      ) {
        this.start = this.$dateUtil.validateDateRange(
          new Date(value[0]),
          this.minDate,
          this.maxDate
        );
        this.end = this.$dateUtil.validateDateRange(
          new Date(value[1]),
          this.minDate,
          this.maxDate
        );
        this.changeLeftMonth({
          month: this.start.getMonth() + 1,
          year: this.start.getFullYear()
        });
      } else {
        this.start = null;
        this.end = null;
      }

      this.onSelect();

      if (this.autoApply) this.clickedApply();
    },
    onUpdateStartTime(value) {
      let start = new Date(this.start);
      start.setHours(value.hours);
      start.setMinutes(value.minutes);
      start.setSeconds(value.seconds);

      this.start = this.$dateUtil.validateDateRange(
        start,
        this.minDate,
        this.maxDate
      );

      // if autoapply is ON we should update the value on time selection change
      if (this.autoApply) {
        this.$emit("update", {
          startDate: this.start,
          endDate:
            this.singleDatePicker && this.singleDatePicker !== "range"
              ? this.start
              : this.end
        });
      }
    },
    onUpdateEndTime(value) {
      let end = new Date(this.end);
      end.setHours(value.hours);
      end.setMinutes(value.minutes);
      end.setSeconds(value.seconds);

      this.end = this.$dateUtil.validateDateRange(
        end,
        this.minDate,
        this.maxDate
      );

      // if autoapply is ON we should update the value on time selection change
      if (this.autoApply) {
        this.$emit("update", { startDate: this.start, endDate: this.end });
      }
    },
    handleEscape(e) {
      if (this.open && e.keyCode === 27 && this.closeOnEsc) {
        this.clickCancel();
      }
    }
  },
  computed: {
    showRanges() {
      return this.ranges !== false && !this.readonly;
    },
    showCalendars() {
      return this.alwaysShowCalendars || this.showCustomRangeCalendars;
    },
    startText() {
      if (this.start === null) return "";
      return this.$dateUtil.format(this.start, this.locale.format);
    },
    endText() {
      if (this.end === null) return "";
      return this.$dateUtil.format(this.end, this.locale.format);
    },
    rangeText() {
      let range = this.startText;
      if (!this.singleDatePicker || this.singleDatePicker === "range") {
        range += this.locale.separator + this.endText;
      }
      return range;
    },
    min() {
      return this.minDate ? new Date(this.minDate) : null;
    },
    max() {
      return this.maxDate ? new Date(this.maxDate) : null;
    },
    pickerStyles() {
      return {
        "show-calendar": this.open || this.opens === "inline",
        "show-ranges": this.showRanges,
        "show-weeknumbers": this.showWeekNumbers,
        single: this.singleDatePicker,
        ["opens" + this.opens]: true,
        linked: this.linkedCalendars,
        "hide-calendars": !this.showCalendars
      };
    },
    isClear() {
      return !this.dateRange.startDate || !this.dateRange.endDate;
    },
    isDirty() {
      let origStart = new Date(this.dateRange.startDate);
      let origEnd = new Date(this.dateRange.endDate);

      return (
        !this.isClear &&
        (this.start.getTime() !== origStart.getTime() ||
          this.end.getTime() !== origEnd.getTime())
      );
    }
  },
  watch: {
    minDate() {
      let dt = this.$dateUtil.validateDateRange(
        this.monthDate,
        this.minDate || new Date(),
        this.maxDate
      );
      this.changeLeftMonth({
        year: dt.getFullYear(),
        month: dt.getMonth() + 1
      });
    },
    maxDate() {
      let dt = this.$dateUtil.validateDateRange(
        this.nextMonthDate,
        this.minDate,
        this.maxDate || new Date()
      );
      this.changeRightMonth({
        year: dt.getFullYear(),
        month: dt.getMonth() + 1
      });
    },
    "dateRange.startDate"(value) {
      if (!this.$dateUtil.isValidDate(new Date(value))) return;

      this.start =
        !!value && !this.isClear && this.$dateUtil.isValidDate(new Date(value))
          ? new Date(value)
          : null;
      if (this.isClear) {
        this.start = null;
        this.end = null;
      } else {
        this.start = new Date(this.dateRange.startDate);
        this.end = new Date(this.dateRange.endDate);
      }
    },
    "dateRange.endDate"(value) {
      if (!this.$dateUtil.isValidDate(new Date(value))) return;

      this.end = !!value && !this.isClear ? new Date(value) : null;
      if (this.isClear) {
        this.start = null;
        this.end = null;
      } else {
        this.start = new Date(this.dateRange.startDate);
        this.end = new Date(this.dateRange.endDate);
      }
    },
    open: {
      handler(value) {
        if (typeof document === "object") {
          this.$nextTick(() => {
            value
              ? document.body.addEventListener("click", this.clickAway)
              : document.body.removeEventListener("click", this.clickAway);
            value
              ? document.addEventListener("keydown", this.handleEscape)
              : document.removeEventListener("keydown", this.handleEscape);

            if (!this.alwaysShowCalendars && this.ranges) {
              this.showCustomRangeCalendars = !Object.keys(this.ranges).find(
                key =>
                  this.$dateUtil.isSame(
                    this.start,
                    this.ranges[key][0],
                    "date"
                  ) &&
                  this.$dateUtil.isSame(this.end, this.ranges[key][1], "date")
              );
            }
          });
        }
      },
      immediate: true
    }
  }
};
</script>
