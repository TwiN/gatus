<template>
  <div v-if="retrievedConfig" class="container container-xs relative mx-auto xl:rounded xl:border xl:shadow-xl xl:my-5 p-5 pb-12 xl:pb-5 text-left dark:bg-gray-800 dark:text-gray-200 dark:border-gray-500" id="global">
    <div class="mb-2">
      <div class="flex flex-wrap">
        <div class="w-3/4 text-left my-auto">
          <div class="text-3xl xl:text-5xl lg:text-4xl font-light">Health Status</div>
        </div>
        <div class="w-1/4 flex justify-end">
          <img v-if="getLogo" :src="getLogo" alt="Gatus" class="object-scale-down" style="max-width: 100px; min-width: 50px; min-height:50px;"/>
          <img v-else src="./assets/logo.svg" alt="Gatus" class="object-scale-down" style="max-width: 100px; min-width: 50px; min-height:50px;"/>
        </div>
      </div>
    </div>
    <div v-if="$route && $route.query.error" class="text-red-500 text-center my-2">
      <div class="text-xl">
        <span class="text-red-500" v-if="$route.query.error === 'access_denied'">You do not have access to this status page</span>
        <span class="text-red-500" v-else>{{ $route.query.error }}</span>
      </div>
    </div>
    <div v-if="config && config.oidc && !config.authenticated">
      <a :href="`${SERVER_URL}/oidc/login`" class="max-w-lg mx-auto w-full flex justify-center py-3 px-4 border border-transparent rounded-md shadow-lg text-white bg-green-700 hover:bg-green-800">
        Login with OIDC
      </a>
    </div>
    <router-view @showTooltip="showTooltip"/>
  </div>
  <Tooltip :result="tooltip.result" :event="tooltip.event"/>
  <Social/>
</template>


<script>
import Social from './components/Social.vue'
import Tooltip from './components/Tooltip.vue';
import {SERVER_URL} from "@/main";

export default {
  name: 'App',
  components: {
    Social,
    Tooltip
  },
  methods: {
    fetchConfig() {
      fetch(`${SERVER_URL}/api/v1/config`, {credentials: 'include'})
      .then(response => {
        this.retrievedConfig = true;
        if (response.status === 200) {
          response.json().then(data => {
            this.config = data;
          })
        }
      });
    },
    showTooltip(result, event) {
      this.tooltip = {result: result, event: event};
    }
  },
  computed: {
    getLogo() {
      return window.config && window.config.logo && window.config.logo !== '{{ .Logo }}' ? window.config.logo : "";
    }
  },
  data() {
    return {
      error: '',
      retrievedConfig: false,
      config: { oidc: false, authenticated: true },
      tooltip: {},
      SERVER_URL
    }
  },
  created() {
    this.fetchConfig();
  }
}
</script>