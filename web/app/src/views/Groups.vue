<template>
  <Loading v-if="!retrievedData" class="h-64 w-64 px-4 my-24"/>
  <slot>
    <div v-for="group in groups" :key="group" class="mb-8">
      <router-link :to="'/groups/' + group" class="text-2xl font-mono text-gray-400">{{ group }}</router-link>
    </div>

  </slot>
  <Settings @refreshData="fetchData"/>
</template>

<script>
import Settings from '@/components/Settings.vue'
import Loading from "@/components/Loading";
import {SERVER_URL} from "@/main.js";

export default {
  name: 'Groups',
  components: {
    Loading,
    Settings,
  },
  emits: ['showTooltip', 'toggleShowAverageResponseTime'],
  methods: {
    fetchData() {
      fetch(`${SERVER_URL}/api/v1/groups`, {credentials: 'include'})
      .then(response => {
        this.retrievedData = true;
        if (response.status === 200) {
          response.json().then(data => {
            if (JSON.stringify(this.groups) !== JSON.stringify(data)) {
              this.groups = data;
            }
          });
        } else {
          response.text().then(text => {
            console.log(`[Groups][fetchData] Error: ${text}`);
          });
        }
      });
    },
    changePage(page) {
      this.retrievedData = false; // Show loading only on page change or on initial load
      this.currentPage = page;
      this.fetchData();
    },
    showTooltip(result, event) {
      this.$emit('showTooltip', result, event);
    },
    toggleShowAverageResponseTime() {
      this.showAverageResponseTime = !this.showAverageResponseTime;
    }
  },
  data() {
    return {
      retrievedData: false,
      groups: [],
      showAverageResponseTime: false,
    }
  },
  mounted() {
    this.fetchData();
  }
}
</script>

