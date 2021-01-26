<template>
  <Services :serviceStatuses="serviceStatuses" :showStatusOnHover="true" @showTooltip="showTooltip"/>
  <Tooltip :result="tooltip.result" :event="tooltip.event"/>
  <Social/>
  <Settings @refreshStatuses="fetchStatuses"/>
</template>


<script>
import Social from './components/Social.vue'
import Settings from './components/Settings.vue'
import Services from './components/Services.vue';
import Tooltip from './components/Tooltip.vue';
import {SERVER_URL} from "./main.js";

export default {
  name: 'App',
  components: {
    Services,
    Social,
    Settings,
    Tooltip
  },
  methods: {
    fetchStatuses() {
      console.log("[App][fetchStatuses] Fetching statuses");
      fetch(`${SERVER_URL}/api/v1/statuses`)
          .then(response => response.json())
          .then(data => {
            if (JSON.stringify(this.serviceStatuses) !== JSON.stringify(data)) {
              console.log(data);
              this.serviceStatuses = data;
            }
          });
    },
    showTooltip(result, event) {
      this.tooltip = {result: result, event: event};
    }
  },
  data() {
    return {
      serviceStatuses: {},
      tooltip: {}
    }
  },
  created() {
    this.fetchStatuses();
  }
}
</script>


<style>
html, body {
  background-color: #f7f9fb;
}

html, body {
  height: 100%;
}
</style>
