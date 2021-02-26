<template>
  <div class='service px-3 py-3 border-l border-r border-t rounded-none hover:bg-gray-100' v-if="data">
    <div class='flex flex-wrap mb-2'>
      <div class='w-3/4'>
        <router-link :to="generatePath()" class="font-bold hover:text-blue-800 hover:underline" title="View detailed service health">
          {{ data.name }}
        </router-link>
        <span v-if="data.results && data.results.length" class='text-gray-500 font-light'> | {{ data.results[data.results.length - 1].hostname }}</span>
      </div>
      <div class='w-1/4 text-right'>
        <span class='font-light status-min-max-ms' v-if="data.results && data.results.length">
          {{ (minResponseTime === maxResponseTime ? minResponseTime : (minResponseTime + '-' + maxResponseTime)) }}ms
        </span>
      </div>
    </div>
    <div>
      <div class='status-over-time flex flex-row'>
        <slot v-if="data.results && data.results.length">
          <slot v-if="data.results.length < maximumNumberOfResults">
            <span v-for="filler in maximumNumberOfResults - data.results.length" :key="filler" class="status rounded border border-dashed">&nbsp;</span>
          </slot>
          <slot v-for="result in data.results" :key="result">
            <span v-if="result.success" class="status status-success rounded bg-success" @mouseenter="showTooltip(result, $event)" @mouseleave="showTooltip(null, $event)"></span>
            <span v-else class="status status-failure rounded bg-red-600" @mouseenter="showTooltip(result, $event)" @mouseleave="showTooltip(null, $event)"></span>
          </slot>
        </slot>
        <slot v-else>
          <span v-for="filler in maximumNumberOfResults" :key="filler" class="status rounded border border-dashed">&nbsp;</span>
        </slot>
      </div>
    </div>
    <div class='flex flex-wrap status-time-ago'>
      <slot v-if="data.results && data.results.length">
        <div class='w-1/2'>
          {{ generatePrettyTimeAgo(data.results[0].timestamp) }}
        </div>
        <div class='w-1/2 text-right'>
          {{ generatePrettyTimeAgo(data.results[data.results.length - 1].timestamp) }}
        </div>
      </slot>
      <slot v-else>
        <div class='w-1/2'>
          &nbsp;
        </div>
      </slot>
    </div>
  </div>
</template>


<script>
import {helper} from "@/mixins/helper";

export default {
  name: 'Service',
  props: {
    maximumNumberOfResults: Number,
    data: Object,
  },
  emits: ['showTooltip'],
  mixins: [helper],
  methods: {
    updateMinAndMaxResponseTimes() {
      let minResponseTime = null;
      let maxResponseTime = null;
      for (let i in this.data.results) {
        const responseTime = parseInt((this.data.results[i].duration/1000000).toFixed(0));
        if (minResponseTime == null || minResponseTime > responseTime) {
          minResponseTime = responseTime;
        }
        if (maxResponseTime == null || maxResponseTime < responseTime) {
          maxResponseTime = responseTime;
        }
      }
      if (this.minResponseTime !== minResponseTime) {
        this.minResponseTime = minResponseTime;
      }
      if (this.maxResponseTime !== maxResponseTime) {
        this.maxResponseTime = maxResponseTime;
      }
    },
    generatePath() {
      if (!this.data) {
        return '/';
      }
      return `/services/${this.data.key}`;
    },
    showTooltip(result, event) {
      this.$emit('showTooltip', result, event);
    }
  },
  watch: {
    data: function () {
      this.updateMinAndMaxResponseTimes();
    }
  },
  created() {
    this.updateMinAndMaxResponseTimes()
  },
  data() {
    return {
      minResponseTime: 0,
      maxResponseTime: 0
    }
  }
}
</script>


<style>
.service:first-child {
  border-top-left-radius: 3px;
  border-top-right-radius: 3px;
}

.service:last-child {
  border-bottom-left-radius: 3px;
  border-bottom-right-radius: 3px;
  border-bottom-width: 3px;
  border-color: #dee2e6;
  border-style: solid;
}

.status-over-time {
  overflow: auto;
}

.status-over-time > span:not(:first-child) {
  margin-left: 2px;
}

.status {
  cursor: pointer;
  transition: all 500ms ease-in-out;
  overflow-x: hidden;
  color: white;
  width: 5%;
  font-size: 75%;
  font-weight: 700;
  text-align: center;
}

.status:hover {
  opacity: 0.7;
  transition: opacity 100ms ease-in-out;
  color: black;
}

.status-time-ago {
  color: #6a737d;
  opacity: 0.5;
  margin-top: 5px;
}

.status-min-max-ms {
  overflow-x: hidden;
}

.status.status-success::after {
  content: "✓";
}

.status.status-failure::after {
  content: "X";
}

@media screen and (max-width: 600px) {
  .status.status-success::after,
  .status.status-failure::after {
    content: " ";
    white-space: pre;
  }
}
</style>