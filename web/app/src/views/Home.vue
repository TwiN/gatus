<template>
  <Services
      :serviceStatuses="serviceStatuses"
      :showStatusOnHover="true"
      @showTooltip="showTooltip"
      @toggleShowAverageResponseTime="toggleShowAverageResponseTime" :showAverageResponseTime="showAverageResponseTime"
  />
  <Pagination @page="changePage"/>
  <Settings @refreshData="fetchData"/>
</template>

<script>
import Settings from '@/components/Settings.vue'
import Services from '@/components/Services.vue';
import Pagination from "@/components/Pagination";
import {SERVER_URL} from "@/main.js";

export default {
  name: 'Home',
  components: {
    Pagination,
    Services,
    Settings,
  },
  emits: ['showTooltip', 'toggleShowAverageResponseTime'],
  methods: {
    fetchData() {
      //console.log("[Home][fetchData] Fetching data");
      fetch(`${SERVER_URL}/api/v1/services/statuses?page=${this.currentPage}`)
          .then(response => response.json())
          .then(data => {
            if (JSON.stringify(this.serviceStatuses) !== JSON.stringify(data)) {
              this.serviceStatuses = data;
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
      serviceStatuses: {},
      currentPage: 1,
      showAverageResponseTime: true
    }
  },
  created() {
    this.fetchData();
  }
}
</script>