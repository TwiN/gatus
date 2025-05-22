<template>
  <router-link to="../"
               class="absolute top-2 left-5 inline-block px-2 pb-0.5 text-sm text-black bg-gray-100 rounded hover:bg-gray-200 focus:outline-none border border-gray-200 dark:bg-gray-700 dark:text-gray-200 dark:border-gray-500 dark:hover:bg-gray-600">
    &larr;
  </router-link>
  <div>
    <slot v-if="endpointStatus">
      <h1 class="text-xl xl:text-3xl font-mono text-gray-400">RECENT CHECKS</h1>
      <hr class="mb-4"/>
      <Endpoint
          :data="endpointStatus"
          :maximumNumberOfResults="20"
          @showTooltip="showTooltip"
          @toggleShowAverageResponseTime="toggleShowAverageResponseTime"
          :showAverageResponseTime="showAverageResponseTime"
      />
      <Pagination @page="changePage" :numberOfResultsPerPage="20" />
    </slot>
    <div v-if="endpointStatus && endpointStatus.key" class="mt-12">
      <h1 class="text-xl xl:text-3xl font-mono text-gray-400">UPTIME</h1>
      <hr/>
      <div class="flex space-x-4 text-center text-2xl mt-6 relative bottom-2 mb-10">
        <div class="flex-1">
          <h2 class="text-sm text-gray-400 mb-1">Last 30 days</h2>
          <img :src="generateUptimeBadgeImageURL('30d')" alt="30d uptime badge" class="mx-auto"/>
        </div>
        <div class="flex-1">
          <h2 class="text-sm text-gray-400 mb-1">Last 7 days</h2>
          <img :src="generateUptimeBadgeImageURL('7d')" alt="7d uptime badge" class="mx-auto"/>
        </div>
        <div class="flex-1">
          <h2 class="text-sm text-gray-400 mb-1">Last 24 hours</h2>
          <img :src="generateUptimeBadgeImageURL('24h')" alt="24h uptime badge" class="mx-auto"/>
        </div>
        <div class="flex-1">
          <h2 class="text-sm text-gray-400 mb-1">Last hour</h2>
          <img :src="generateUptimeBadgeImageURL('1h')" alt="1h uptime badge" class="mx-auto"/>
        </div>
      </div>
    </div>
    <div v-if="endpointStatus && endpointStatus.key && showResponseTimeChartAndBadges" class="mt-12">
      <div class="flex items-center justify-between">
        <h1 class="text-xl xl:text-3xl font-mono text-gray-400">RESPONSE TIME</h1>
        <select v-model="selectedChartDuration"  class="text-sm bg-gray-400 text-white border border-gray-600 rounded-md px-3 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500">
          <option value="24h">24 hours</option>
          <option value="7d">7 days</option>
          <option value="30d">30 days</option>
        </select>
      </div>
      <img :src="generateResponseTimeChartImageURL(selectedChartDuration)" alt="response time chart" class="mt-6"/>
      <div class="flex space-x-4 text-center text-2xl mt-6 relative bottom-2 mb-10">
        <div class="flex-1">
          <h2 class="text-sm text-gray-400 mb-1">Last 30 days</h2>
          <img :src="generateResponseTimeBadgeImageURL('30d')" alt="7d response time badge" class="mx-auto mt-2"/>
        </div>
        <div class="flex-1">
          <h2 class="text-sm text-gray-400 mb-1">Last 7 days</h2>
          <img :src="generateResponseTimeBadgeImageURL('7d')" alt="7d response time badge" class="mx-auto mt-2"/>
        </div>
        <div class="flex-1">
          <h2 class="text-sm text-gray-400 mb-1">Last 24 hours</h2>
          <img :src="generateResponseTimeBadgeImageURL('24h')" alt="24h response time badge" class="mx-auto mt-2"/>
        </div>
        <div class="flex-1">
          <h2 class="text-sm text-gray-400 mb-1">Last hour</h2>
          <img :src="generateResponseTimeBadgeImageURL('1h')" alt="1h response time badge" class="mx-auto mt-2"/>
        </div>
      </div>
    </div>
    <div v-if="endpointStatus && endpointStatus.key">
      <h1 class="text-xl xl:text-3xl font-mono text-gray-400 mt-4">CURRENT HEALTH</h1>
      <hr />
      <div class="flex space-x-4 text-center text-2xl mt-6 relative bottom-2 mb-10">
        <div class="flex-1">
          <img :src="generateHealthBadgeImageURL()" alt="health badge" class="mx-auto"/>
        </div>
      </div>
    </div>
    <div v-if="endpointStatus && endpointStatus.key">
      <h1 class="text-xl xl:text-3xl font-mono text-gray-400 mt-4">EVENTS</h1>
      <hr />
      <ul role="list" class="px-0 xl:px-24 divide-y divide-gray-200 dark:divide-gray-600">
        <li v-for="event in events" :key="event" class="p-3 my-4">
          <h2 class="text-sm sm:text-lg">
            <ArrowUpCircleIcon v-if="event.type === 'HEALTHY'" class="w-8 inline mr-2 text-green-600" />
            <ArrowDownCircleIcon v-else-if="event.type === 'UNHEALTHY'" class="w-8 inline mr-2 text-red-500" />
            <PlayCircleIcon v-else-if="event.type === 'START'" class="w-8 inline mr-2 text-gray-400 dark:text-gray-100" />
            {{ event.fancyText }}
          </h2>
          <div class="flex mt-1 text-xs sm:text-sm text-gray-400">
            <div class="flex-2 text-left pl-12">
              {{ prettifyTimestamp(event.timestamp) }}
            </div>
            <div class="flex-1 text-right">
              {{ event.fancyTimeAgo }}
            </div>
          </div>
        </li>
      </ul>
    </div>
  </div>
  <Settings @refreshData="fetchData"/>
</template>


<script>
import Settings from '@/components/Settings.vue'
import Endpoint from '@/components/Endpoint.vue';
import {SERVER_URL} from "@/main.js";
import {helper} from "@/mixins/helper.js";
import Pagination from "@/components/Pagination";
import { ArrowDownCircleIcon, ArrowUpCircleIcon, PlayCircleIcon } from '@heroicons/vue/20/solid'

export default {
  name: 'Details',
  components: {
    Pagination,
    Endpoint,
    Settings,
    ArrowDownCircleIcon,
    ArrowUpCircleIcon,
    PlayCircleIcon
  },
  emits: ['showTooltip'],
  mixins: [helper],
  methods: {
    fetchData() {
      //console.log("[Details][fetchData] Fetching data");
      fetch(`${this.serverUrl}/api/v1/endpoints/${this.$route.params.key}/statuses?page=${this.currentPage}`, {credentials: 'include'})
      .then(response => {
        if (response.status === 200) {
          response.json().then(data => {
            if (JSON.stringify(this.endpointStatus) !== JSON.stringify(data)) {
              this.endpointStatus = data;
              let events = [];
              for (let i = data.events.length - 1; i >= 0; i--) {
                let event = data.events[i];
                if (i === data.events.length - 1) {
                  if (event.type === 'UNHEALTHY') {
                    event.fancyText = 'Endpoint is unhealthy';
                  } else if (event.type === 'HEALTHY') {
                    event.fancyText = 'Endpoint is healthy';
                  } else if (event.type === 'START') {
                    event.fancyText = 'Monitoring started';
                  }
                } else {
                  let nextEvent = data.events[i + 1];
                  if (event.type === 'HEALTHY') {
                    event.fancyText = 'Endpoint became healthy';
                  } else if (event.type === 'UNHEALTHY') {
                    if (nextEvent) {
                      event.fancyText = 'Endpoint was unhealthy for ' + this.generatePrettyTimeDifference(nextEvent.timestamp, event.timestamp);
                    } else {
                      event.fancyText = 'Endpoint became unhealthy';
                    }
                  } else if (event.type === 'START') {
                    event.fancyText = 'Monitoring started';
                  }
                }
                event.fancyTimeAgo = this.generatePrettyTimeAgo(event.timestamp);
                events.push(event);
              }
              this.events = events;
              // Check if there's any non-0 response time data
              // If there isn't, it's likely an external endpoint, which means we should
              // hide the response time chart and badges
              for (let i = 0; i < data.results.length; i++) {
                if (data.results[i].duration > 0) {
                  this.showResponseTimeChartAndBadges = true;
                  break;
                }
              }
            }
          });
        } else {
          response.text().then(text => {
            console.log(`[Details][fetchData] Error: ${text}`);
          });
        }
      });
    },
    generateHealthBadgeImageURL() {
      return `${this.serverUrl}/api/v1/endpoints/${this.endpointStatus.key}/health/badge.svg`;
    },
    generateUptimeBadgeImageURL(duration) {
      return `${this.serverUrl}/api/v1/endpoints/${this.endpointStatus.key}/uptimes/${duration}/badge.svg`;
    },
    generateResponseTimeBadgeImageURL(duration) {
      return `${this.serverUrl}/api/v1/endpoints/${this.endpointStatus.key}/response-times/${duration}/badge.svg`;
    },
    generateResponseTimeChartImageURL(duration) {
      return `${this.serverUrl}/api/v1/endpoints/${this.endpointStatus.key}/response-times/${duration}/chart.svg`;
    },
    changePage(page) {
      this.currentPage = page;
      this.fetchData();
    },
    showTooltip(result, event) {
      this.$emit('showTooltip', result, event);
    },
    toggleShowAverageResponseTime() {
      this.showAverageResponseTime = !this.showAverageResponseTime;
    },
  },
  data() {
    return {
      endpointStatus: {},
      events: [],
      hourlyAverageResponseTime: {},
      selectedChartDuration: '24h',
      // Since this page isn't at the root, we need to modify the server URL a bit
      serverUrl: SERVER_URL === '.' ? '..' : SERVER_URL,
      currentPage: 1,
      showAverageResponseTime: true,
      showResponseTimeChartAndBadges: false,
      chartLabels: [],
      chartValues: [],
    }
  },
  created() {
    this.fetchData();
  }
}
</script>

<style scoped>
.endpoint {
  border-radius: 3px;
  border-bottom-width: 3px;
}
</style>