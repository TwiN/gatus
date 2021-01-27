<template>
  <router-link to="/" class="absolute top-2 left-2 inline-block px-2 py-0 text-lg text-black transition bg-gray-100 rounded shadow ripple hover:shadow-lg hover:bg-gray-200 focus:outline-none">
    &#8592;
  </router-link>
  <div class="container mx-auto">
    <slot v-if="serviceStatus">
      <h1 class="text-3xl text-monospace text-gray-400">RECENT CHECKS</h1>
      <hr class="mb-4" />
      <Service :data="serviceStatus" :maximumNumberOfResults="20" @showTooltip="showTooltip" />
    </slot>
    <div v-if="serviceStatus.uptime" class="mt-5">
      <h1 class="text-3xl text-monospace text-gray-400">UPTIME</h1>
      <hr />
      <div class="flex space-x-4 text-center text-2xl mt-5">
        <div class="flex-1">
          {{ prettifyUptime(serviceStatus.uptime['7d']) }}
          <h2 class="text-sm text-gray-400">Last 7 days</h2>
        </div>
        <div class="flex-1">
          {{ prettifyUptime(serviceStatus.uptime['24h']) }}
          <h2 class="text-sm text-gray-400">Last 24 hours</h2>
        </div>
        <div class="flex-1">
          {{ prettifyUptime(serviceStatus.uptime['1h']) }}
          <h2 class="text-sm text-gray-400">Last hour</h2>
        </div>
      </div>
      <h3 class="text-xl text-monospace text-gray-400">BADGES</h3>
      <hr />
      <div class="flex space-x-4 text-center text-2xl mt-5">
        <div class="flex-1">
          <img :src="generateBadgeImageURL('7d')" alt="7d uptime badge" class="mx-auto" />
        </div>
        <div class="flex-1">
          <img :src="generateBadgeImageURL('24h')" alt="7d uptime badge" class="mx-auto" />
        </div>
        <div class="flex-1">
          <img :src="generateBadgeImageURL('1h')" alt="7d uptime badge" class="mx-auto" />
        </div>
      </div>
    </div>
  </div>
  <Settings @refreshData="fetchData"/>
</template>


<script>
import Settings from '@/components/Settings.vue'
import Service from '@/components/Service.vue';
import {SERVER_URL} from "@/main.js";

export default {
  name: 'Details',
  components: {
    Service,
    Settings,
  },
  emits: ['showTooltip'],
  methods: {
    fetchData() {
      console.log("[Details][fetchData] Fetching data");
      fetch(`${SERVER_URL}/api/v1/statuses/${this.$route.params.key}`)
          .then(response => response.json())
          .then(data => {
            if (JSON.stringify(this.serviceStatus) !== JSON.stringify(data)) {
              console.log(data);
              this.serviceStatus = data;
            }
          });
    },
    generateBadgeImageURL(duration) {
      return `${SERVER_URL}/api/v1/badges/uptime/${duration}/${this.serviceStatus.key}`;
    },
    prettifyUptime(uptime) {
      if (!uptime) {
        return "0%";
      }
      return (uptime * 100).toFixed(2) + "%"
    },
    showTooltip(result, event) {
      this.$emit('showTooltip', result, event);
    }
  },
  data() {
    return {
      serviceStatus: {}
    }
  },
  created() {
    this.fetchData();
  }
}
</script>

<style scoped>
.service {
  border-bottom-left-radius: 3px;
  border-bottom-right-radius: 3px;
  border-bottom-width: 1px;
  border-color: #dee2e6;
  border-style: solid;
}
</style>