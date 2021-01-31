<template>
  <div class='service px-3 py-3 border-l border-r border-t rounded-none hover:bg-gray-100' v-if="data && data.results && data.results.length">
    <div class='flex flex-wrap mb-2'>
      <div class='w-3/4'>
        <router-link :to="generatePath()" class="font-bold hover:text-blue-800 hover:underline" title="View detailed service health">
          {{ data.name }}
        </router-link>
        <span class='text-gray-500 font-light'> | {{ data.results[data.results.length - 1].hostname }}</span>
      </div>
      <div class='w-1/4 text-right'>
        <span class='font-light status-min-max-ms'>
          {{ (minResponseTime === maxResponseTime ? minResponseTime : (minResponseTime + '-' + maxResponseTime)) }}ms
        </span>
      </div>
    </div>
    <div>
      <div class='status-over-time flex flex-row'>
        <slot v-for="filler in maximumNumberOfResults - data.results.length" :key="filler">
          <span class="status rounded border border-dashed"> </span>
        </slot>
        <slot v-for="result in data.results" :key="result">
          <span v-if="result.success" class="status rounded bg-success" @mouseenter="showTooltip(result, $event)" @mouseleave="showTooltip(null, $event)">&#10003;</span>
          <span v-else class="status rounded bg-red-600" @mouseenter="showTooltip(result, $event)" @mouseleave="showTooltip(null, $event)">X</span>
        </slot>
      </div>
    </div>
    <div class='flex flex-wrap status-time-ago'>
      <!-- Show "Last update at" instead? -->
      <div class='w-1/2'>
        {{ generatePrettyTimeAgo(data.results[0].timestamp) }}
      </div>
      <div class='w-1/2 text-right'>
        {{ generatePrettyTimeAgo(data.results[data.results.length - 1].timestamp) }}
      </div>
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
        const responseTime = parseInt(this.data.results[i].duration/1000000);
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

.status-over-time {
  overflow: auto;
}

.status-over-time > span:not(:first-child) {
  margin-left: 2px;
}

.status-time-ago {
  color: #6a737d;
  opacity: 0.5;
  margin-top: 5px;
}

.status-min-max-ms {
  overflow-x: hidden;
}
</style>