<template>
  <router-link to="../" class="absolute top-2 left-2 inline-block px-2 py-0 text-lg text-black transition bg-gray-100 rounded shadow ripple hover:shadow-lg hover:bg-gray-200 focus:outline-none">
    &larr;
  </router-link>
  <div>
    <slot v-if="serviceStatus">
      <h1 class="text-xl xl:text-3xl text-monospace text-gray-400">RECENT CHECKS</h1>
      <hr class="mb-4" />
      <Service :data="serviceStatus" :maximumNumberOfResults="20" @showTooltip="showTooltip" />
    </slot>
    <div v-if="uptime" class="mt-12">
      <h1 class="text-xl xl:text-3xl text-monospace text-gray-400">UPTIME</h1>
      <hr />
      <div class="flex space-x-4 text-center text-xl xl:text-2xl mt-3">
        <div class="flex-1">
          {{ prettifyUptime(uptime['7d']) }}
          <h2 class="text-sm text-gray-400">Last 7 days</h2>
        </div>
        <div class="flex-1">
          {{ prettifyUptime(uptime['24h']) }}
          <h2 class="text-sm text-gray-400">Last 24 hours</h2>
        </div>
        <div class="flex-1">
          {{ prettifyUptime(uptime['1h']) }}
          <h2 class="text-sm text-gray-400">Last hour</h2>
        </div>
      </div>
      <hr class="mt-1"/>
      <h3 class="text-xl text-monospace text-gray-400 mt-1 text-right">BADGES</h3>
      <div v-if="serviceStatus && serviceStatus.key" class="flex space-x-4 text-center text-2xl mt-6 relative bottom-12">
        <div class="flex-1">
          <img :src="generateBadgeImageURL('7d')" alt="7d uptime badge" class="mx-auto" />
        </div>
        <div class="flex-1">
          <img :src="generateBadgeImageURL('24h')" alt="24h uptime badge" class="mx-auto" />
        </div>
        <div class="flex-1">
          <img :src="generateBadgeImageURL('1h')" alt="1h uptime badge" class="mx-auto" />
        </div>
      </div>
    </div>
    <div>
      <h1 class="text-xl xl:text-3xl text-monospace text-gray-400 mt-4">EVENTS</h1>
      <hr class="mb-4" />
      <div>
        <slot v-for="event in events" :key="event">
          <div class="p-3 my-4">
            <h2 class="text-lg">
              <img v-if="event.type === 'HEALTHY'" src="../assets/arrow-up-green.png" alt="Healthy" class="border border-green-600 rounded-full opacity-75 bg-green-100 mr-2 inline" width="26" />
              <img v-else-if="event.type === 'UNHEALTHY'" src="../assets/arrow-down-red.png" alt="Unhealthy" class="border border-red-500 rounded-full opacity-75 bg-red-100 mr-2 inline" width="26" />
              <img v-else-if="event.type === 'START'" src="../assets/arrow-right-black.png" alt="Start" class="border border-gray-500 rounded-full opacity-75 bg-gray-100 mr-2 inline" width="26" />
              {{ event.fancyText }}
            </h2>
            <div class="flex mt-1 text-sm text-gray-400">
              <div class="flex-1 text-left pl-10">
                {{ new Date(event.timestamp).toISOString() }}
              </div>
              <div class="flex-1 text-right">
                {{ event.fancyTimeAgo }}
              </div>
            </div>
          </div>
        </slot>
      </div>
    </div>
  </div>
  <Settings @refreshData="fetchData"/>
</template>


<script>
import Settings from '@/components/Settings.vue'
import Service from '@/components/Service.vue';
import {SERVER_URL} from "@/main.js";
import {helper} from "@/mixins/helper.js";

export default {
  name: 'Details',
  components: {
    Service,
    Settings,
  },
  emits: ['showTooltip'],
  mixins: [helper],
  methods: {
    fetchData() {
      //console.log("[Details][fetchData] Fetching data");
      fetch(`${this.serverUrl}/api/v1/statuses/${this.$route.params.key}`)
          .then(response => response.json())
          .then(data => {
            if (JSON.stringify(this.serviceStatus) !== JSON.stringify(data)) {
              this.serviceStatus = data.serviceStatus;
              this.uptime = data.uptime;
              let events = [];
              for (let i = data.events.length-1; i >= 0; i--) {
                let event = data.events[i];
                if (i === data.events.length-1) {
                  if (event.type === 'UNHEALTHY') {
                    event.fancyText = 'Service is unhealthy';
                  } else if (event.type === 'HEALTHY') {
                    event.fancyText = 'Service is healthy';
                  } else if (event.type === 'START') {
                    event.fancyText = 'Monitoring started';
                  }
                } else {
                  let nextEvent = data.events[i+1];
                  if (event.type === 'HEALTHY') {
                    event.fancyText = 'Service became healthy';
                  } else if (event.type === 'UNHEALTHY') {
                    if (nextEvent) {
                      event.fancyText = 'Service was unhealthy for ' + this.prettifyTimeDifference(nextEvent.timestamp, event.timestamp);
                    } else {
                      event.fancyText = 'Service became unhealthy';
                    }
                  } else if (event.type === 'START') {
                    event.fancyText = 'Monitoring started';
                  }
                }
                event.fancyTimeAgo = this.generatePrettyTimeAgo(event.timestamp);
                events.push(event);
              }
              this.events = events;
            }
          });
    },
    generateBadgeImageURL(duration) {
      return `${this.serverUrl}/api/v1/badges/uptime/${duration}/${this.serviceStatus.key}.svg`;
    },
    prettifyUptime(uptime) {
      if (!uptime) {
        return '0%';
      }
      return (uptime * 100).toFixed(2) + '%'
    },
    prettifyTimeDifference(start, end) {
      let minutes = Math.ceil((new Date(start) - new Date(end))/1000/60);
      return minutes + (minutes === 1 ? ' minute' : ' minutes');
    },
    showTooltip(result, event) {
      this.$emit('showTooltip', result, event);
    }
  },
  data() {
    return {
      serviceStatus: {},
      events: [],
      uptime: {"7d": 0, "24h": 0, "1h": 0},
      // Since this page isn't at the root, we need to modify the server URL a bit
      serverUrl: SERVER_URL === '.' ? '..' : SERVER_URL,
    }
  },
  created() {
    this.fetchData();
  }
}
</script>

<style scoped>
.service {
  border-radius: 3px;
  border-bottom-width: 3px;
  border-color: #dee2e6;
  border-style: solid;
}
</style>