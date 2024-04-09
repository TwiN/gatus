<template>
  <div id="tooltip" ref="tooltip" :class="hidden ? 'invisible' : ''" :style="'top:' + top + 'px; left:' + left + 'px'">
    <slot v-if="result">
      <div class="tooltip-title">Timestamp:</div>
      <code id="tooltip-timestamp">{{ prettifyTimestamp(result.timestamp) }}</code>
      <div class="tooltip-title">Response time:</div>
      <code id="tooltip-response-time">{{ (result.duration / 1000000).toFixed(0) }}ms</code>
      <slot v-if="result.conditionResults && result.conditionResults.length">
        <div class="tooltip-title">Conditions:</div>
        <code id="tooltip-conditions">
          <slot v-for="conditionResult in result.conditionResults" :key="conditionResult">
            {{ conditionResult.success ? "&#10003;" : "X" }} ~ {{ conditionResult.condition }}<br/>
          </slot>
        </code>
      </slot>
      <div id="tooltip-errors-container" v-if="result.errors && result.errors.length">
        <div class="tooltip-title">Errors:</div>
        <code id="tooltip-errors">
          <slot v-for="error in result.errors" :key="error">
            - {{ error }}<br/>
          </slot>
        </code>
      </div>
    </slot>
  </div>
</template>


<script>
import {helper} from "@/mixins/helper";

export default {
  name: 'Endpoints',
  props: {
    event: Event,
    result: Object
  },
  mixins: [helper],
  methods: {
    htmlEntities(s) {
      return String(s)
          .replace(/&/g, '&amp;')
          .replace(/</g, '&lt;')
          .replace(/>/g, '&gt;')
          .replace(/"/g, '&quot;')
          .replace(/'/g, '&apos;');
    },
    reposition() {
      if (this.event && this.event.type) {
        if (this.event.type === 'mouseenter') {
          let targetTopPosition = this.event.target.getBoundingClientRect().y + 30;
          let targetLeftPosition = this.event.target.getBoundingClientRect().x;
          let tooltipBoundingClientRect = this.$refs.tooltip.getBoundingClientRect();
          if (targetLeftPosition + window.scrollX + tooltipBoundingClientRect.width + 50 > document.body.getBoundingClientRect().width) {
            targetLeftPosition = this.event.target.getBoundingClientRect().x - tooltipBoundingClientRect.width + this.event.target.getBoundingClientRect().width;
            if (targetLeftPosition < 0) {
              targetLeftPosition += -targetLeftPosition;
            }
          }
          if (targetTopPosition + window.scrollY + tooltipBoundingClientRect.height + 50 > document.body.getBoundingClientRect().height && targetTopPosition >= 0) {
            targetTopPosition = this.event.target.getBoundingClientRect().y - (tooltipBoundingClientRect.height + 10);
            if (targetTopPosition < 0) {
              targetTopPosition = this.event.target.getBoundingClientRect().y + 30;
            }
          }
          this.top = targetTopPosition;
          this.left = targetLeftPosition;
        } else if (this.event.type === 'mouseleave') {
          this.hidden = true;
        }
      }
    }
  },
  watch: {
    event: function (value) {
      if (value && value.type) {
        if (value.type === 'mouseenter') {
          this.hidden = false;
        } else if (value.type === 'mouseleave') {
          this.hidden = true;
        }
      }
    }
  },
  updated() {
    this.reposition();
  },
  created() {
    this.reposition();
  },
  data() {
    return {
      hidden: true,
      top: 0,
      left: 0
    }
  }
}
</script>


<style>
#tooltip {
  position: fixed;
  background-color: white;
  border: 1px solid lightgray;
  border-radius: 4px;
  padding: 6px;
  font-size: 13px;
}

#tooltip code {
  color: #212529;
  line-height: 1;
}

#tooltip .tooltip-title {
  font-weight: bold;
  margin-bottom: 0;
  display: block;
}

#tooltip .tooltip-title {
  margin-top: 8px;
}

#tooltip > .tooltip-title:first-child {
  margin-top: 0;
}
</style>
