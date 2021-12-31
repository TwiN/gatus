<template>
  <Endpoints
      :endpointStatuses="endpointStatuses"
      :showStatusOnHover="true"
      @showTooltip="showTooltip"
      @toggleShowAverageResponseTime="toggleShowAverageResponseTime" :showAverageResponseTime="showAverageResponseTime"
  />
  <Pagination @page="changePage"/>
  <Settings @refreshData="fetchData"/>
</template>

<script>
import Settings from '@/components/Settings.vue'
import Endpoints from '@/components/Endpoints.vue';
import Pagination from "@/components/Pagination";
import {SERVER_URL} from "@/main.js";

export default {
  name: 'Home',
  components: {
    Pagination,
    Endpoints,
    Settings,
  },
  emits: ['showTooltip', 'toggleShowAverageResponseTime'],
  methods: {
    fetchData() {
      //console.log("[Home][fetchData] Fetching data");
      fetch(`${SERVER_URL}/api/v1/endpoints/statuses?page=${this.currentPage}`, {credentials: 'include'})
          .then(response => response.json())
          .then(data => {
            if (JSON.stringify(this.endpointStatuses) !== JSON.stringify(data)) {
              this.endpointStatuses = data;
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
      showAverageResponseTime: true
    }
  },
  created() {
    this.fetchData();
  }
}
</script>