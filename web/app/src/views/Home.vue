<template>
  <Services :serviceStatuses="serviceStatuses" :showStatusOnHover="true" @showTooltip="showTooltip"/>
  <Settings @refreshData="fetchData"/>
</template>

<script>
import Settings from '@/components/Settings.vue'
import Services from '@/components/Services.vue';
import {SERVER_URL} from "@/main.js";

export default {
  name: 'Home',
  components: {
    Services,
    Settings,
  },
  emits: ['showTooltip'],
  methods: {
    fetchData() {
      //console.log("[Home][fetchData] Fetching data");
      fetch(`${SERVER_URL}/api/v1/statuses`)
          .then(response => response.json())
          .then(data => {
            if (JSON.stringify(this.serviceStatuses) !== JSON.stringify(data)) {
              this.serviceStatuses = data;
            }
          });
    },
    showTooltip(result, event) {
      this.$emit('showTooltip', result, event);
    }
  },
  data() {
    return {
      serviceStatuses: {}
    }
  },
  created() {
    this.fetchData();
  }
}
</script>