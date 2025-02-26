<template>
  <div class='endpoint px-3 py-3 border-l border-r border-t rounded-none hover:bg-gray-100 dark:hover:bg-gray-700 dark:border-gray-500' v-if="data">
    <div class='flex flex-wrap mb-2'>
      <div class='w-3/4'>
        <router-link :to="generatePath()" class="font-bold hover:text-blue-800 hover:underline dark:hover:text-blue-400" title="View detailed endpoint health">
          {{ data.name }}
        </router-link>
        <span v-if="data.results && data.results.length && data.results[data.results.length - 1].hostname" class='text-gray-500 font-light'> | {{ data.results[data.results.length - 1].hostname }}</span>
        <div v-if="data.uiConfig.Menu.length > 0" class="relative inline-block ml-2">
          <div>
            <button type="button" class="relative top-1 inline-flex w-full justify-center gap-x-1.5 rounded-md bg-white pr-1 py-0 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50" id="menu-button" aria-expanded="true" aria-haspopup="true" @click="toggleMenu">
              <svg class="-mr-1 size-5 text-gray-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true" data-slot="icon">
                <path fill-rule="evenodd" d="M5.22 8.22a.75.75 0 0 1 1.06 0L10 11.94l3.72-3.72a.75.75 0 1 1 1.06 1.06l-4.25 4.25a.75.75 0 0 1-1.06 0L5.22 9.28a.75.75 0 0 1 0-1.06Z" clip-rule="evenodd" />
              </svg>
            </button>
          </div>
          <div v-if="menuOpen" class="absolute left-0 z-10 mt-2 w-56 origin-top-left rounded-md bg-white shadow-lg ring-1 ring-black/5 focus:outline-none" role="menu" aria-orientation="vertical" aria-labelledby="menu-button" tabindex="-1">
            <div class="py-1" role="none">
              <template v-for="item in data.uiConfig.Menu" :key="item.Name">
                <a :href="item.Value" v-if="item.Type === 'link'" class="block px-4 py-2 text-sm text-gray-700" role="menuitem" tabindex="-1">{{item.Name}}</a>
                <span v-else class="block px-4 py-2 text-sm text-gray-700">{{item.Name}}</span>
              </template>
            </div>
          </div>
        </div>
      </div>
      <div class='w-1/4 text-right'>
        <span class='font-light overflow-x-hidden cursor-pointer select-none hover:text-gray-500' v-if="data.results && data.results.length" @click="toggleShowAverageResponseTime" :title="showAverageResponseTime ? 'Average response time' : 'Minimum and maximum response time'">
          <slot v-if="showAverageResponseTime">
            ~{{ averageResponseTime }}ms
          </slot>
          <slot v-else>
            {{ (minResponseTime === maxResponseTime ? minResponseTime : (minResponseTime + '-' + maxResponseTime)) }}ms
          </slot>
        </span>
<!--        <span class="text-sm font-bold cursor-pointer">-->
<!--          ⋯-->
<!--        </span>-->
      </div>
    </div>
    <div>
      <div class='status-over-time flex flex-row'>
        <slot v-if="data.results && data.results.length">
          <slot v-if="data.results.length < maximumNumberOfResults">
            <span v-for="filler in maximumNumberOfResults - data.results.length" :key="filler" class="status rounded border border-dashed border-gray-400">&nbsp;</span>
          </slot>
          <slot v-for="result in data.results" :key="result">
            <span v-if="result.success" class="status status-success rounded bg-success" @mouseenter="showTooltip(result, $event)" @mouseleave="showTooltip(null, $event)"></span>
            <span v-else class="status status-failure rounded bg-red-600" @mouseenter="showTooltip(result, $event)" @mouseleave="showTooltip(null, $event)"></span>
          </slot>
        </slot>
        <slot v-else>
          <span v-for="filler in maximumNumberOfResults" :key="filler" class="status rounded border border-dashed border-gray-400">&nbsp;</span>
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
  name: 'Endpoint',
  props: {
    maximumNumberOfResults: Number,
    data: Object,
    showAverageResponseTime: Boolean
  },
  emits: ['showTooltip', 'toggleShowAverageResponseTime'],
  mixins: [helper],
  methods: {
    updateMinAndMaxResponseTimes() {
      let minResponseTime = null;
      let maxResponseTime = null;
      let totalResponseTime = 0;
      for (let i in this.data.results) {
        const responseTime = parseInt((this.data.results[i].duration/1000000).toFixed(0));
        totalResponseTime += responseTime;
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
      if (this.data.results && this.data.results.length) {
        this.averageResponseTime = (totalResponseTime/this.data.results.length).toFixed(0);
      }
    },
    generatePath() {
      if (!this.data) {
        return '/';
      }
      return `/endpoints/${this.data.key}`;
    },
    showTooltip(result, event) {
      this.$emit('showTooltip', result, event);
    },
    toggleShowAverageResponseTime() {
      this.$emit('toggleShowAverageResponseTime');
    },
    toggleMenu() {
      this.menuOpen = !this.menuOpen;
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
      maxResponseTime: 0,
      averageResponseTime: 0,
      menuOpen: false,
    }
  }
}
</script>


<style>
.endpoint:first-child {
  border-top-left-radius: 3px;
  border-top-right-radius: 3px;
}

.endpoint:last-child {
  border-bottom-left-radius: 3px;
  border-bottom-right-radius: 3px;
  border-bottom-width: 3px;
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

.size-5 {
  --tw-spacing: 0.25rem;
	width: calc(var(--tw-spacing) * 5);
	height: calc(var(--tw-spacing) * 5);
}
</style>
