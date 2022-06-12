<template>
  <Loading v-if="!retrievedData" class="h-64 w-64 px-4 my-24"/>
  <slot v-else>
    <Endpoints
        :endpointStatuses="endpointStatuses"
        :showStatusOnHover="true"
        @showTooltip="showTooltip"
        @toggleShowAverageResponseTime="toggleShowAverageResponseTime"
        :showAverageResponseTime="showAverageResponseTime"
    />
    <Pagination @page="changePage"/>
  </slot>
  <Settings @refreshData="fetchData"/>
</template>

<script>
import Settings from '@/components/Settings.vue'
import Endpoints from '@/components/Endpoints.vue';
import Pagination from "@/components/Pagination";
import Loading from "@/components/Loading";
import {SERVER_URL} from "@/main.js";

export default {
  name: 'Home',
  components: {
    Loading,
    Pagination,
    Endpoints,
    Settings,
  },
  emits: ['showTooltip', 'toggleShowAverageResponseTime'],
  methods: {
    fetchData() {
      fetch(`${SERVER_URL}/api/v1/endpoints/statuses?page=${this.currentPage}`, {credentials: 'include'})
      .then(response => {
        this.retrievedData = true;
        if (response.status === 200) {
          response.json().then(data => {
            if (JSON.stringify(this.endpointStatuses) !== JSON.stringify(data)) {
              this.endpointStatuses = data;
            }
          });
        } else {
          response.text().then(text => {
            console.log(`[Home][fetchData] Error: ${text}`);
          });
        }
      });
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
      endpointStatuses: [],
      currentPage: 1,
      showAverageResponseTime: true,
      retrievedData: false,
    }
  },
  created() {
    this.fetchData();
  }
}
</script>