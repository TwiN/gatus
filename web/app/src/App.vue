<template>
  <Services :serviceStatuses="serviceStatuses" :maximumNumberOfResults="20" :showStatusOnHover="true" />
  <Social />
  <Settings @refreshStatuses="fetchStatuses" />
</template>


<script>
import Social from './components/Social.vue'
import Settings from './components/Settings.vue'
import Services from './components/Services.vue';

export default {
  name: 'App',
  components: {
    Services,
    Social,
    Settings
  },
  methods: {
    fetchStatuses() {
      console.log("[App][fetchStatuses] Fetching statuses");
      fetch("http://localhost:8080/api/v1/statuses")
          .then(response => response.json())
          .then(data => {
            if (JSON.stringify(this.serviceStatuses) !== JSON.stringify(data)) {
              console.log(data);
              this.serviceStatuses = data;
            }
          });
    }
  },
  data() {
    return {
      serviceStatuses: {}
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
  html {
    height: 100%;
  }
</style>
