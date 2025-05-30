<template>
  <Loading v-if="!retrievedConfig" class="h-64 w-64 px-4" />
  <div v-else :class="[config && config.oidc && !config.authenticated ? 'hidden' : '', 'container container-xs relative mx-auto xl:rounded xl:border xl:shadow-xl xl:my-5 p-5 pb-12 xl:pb-5 text-left dark:bg-gray-800 dark:text-gray-200 dark:border-gray-500']" id="global">
    <div class="mb-2">
      <div class="flex flex-wrap">
        <div class="w-3/4 text-left my-auto">
          <div class="text-3xl xl:text-5xl lg:text-4xl font-light">{{ header }}</div>
        </div>
        <div class="w-1/4 flex justify-end">
          <component :is="link ? 'a' : 'div'" :href="link" target="_blank" class="flex items-center justify-center" style="width:100px;min-height:100px;">
            <img v-if="logo" :src="logo" alt="Gatus" class="object-scale-down" style="max-width:100px;min-width:50px;min-height:50px;" />
            <img v-else src="./assets/logo.svg" alt="Gatus" class="object-scale-down" style="max-width:100px;min-width:50px;min-height:50px;" />
          </component>
        </div>
      </div>
      <div v-if="buttons" class="flex flex-wrap">
        <a v-for="button in buttons" :key="button.name" :href="button.link" target="_blank" class="px-2 py-0.5 font-medium select-none text-gray-600 hover:text-gray-500 dark:text-gray-300 dark:hover:text-gray-400 hover:underline">
          {{ button.name }}
        </a>
      </div>
    </div>
    <router-view @showTooltip="showTooltip" />
  </div>

  <div v-if="config && config.oidc && !config.authenticated" class="mx-auto max-w-md pt-12">
    <img src="./assets/logo.svg" alt="Gatus" class="mx-auto" style="max-width:160px; min-width:50px; min-height:50px;"/>
    <h2 class="mt-4 text-center text-4xl font-extrabold text-gray-800 dark:text-gray-200">
      Gatus
    </h2>
    <div class="py-7 px-4 rounded-sm sm:px-10">
      <div v-if="$route && $route.query.error" class="text-red-500 text-center mb-5">
        <div class="text-sm">
          <span class="text-red-500" v-if="$route.query.error === 'access_denied'">You do not have access to this status page</span>
          <span class="text-red-500" v-else>{{ $route.query.error }}</span>
        </div>
      </div>
      <div>
        <a :href="`${SERVER_URL}/oidc/login`" class="max-w-lg mx-auto w-full flex justify-center py-3 px-4 border border-green-800 rounded-md shadow-lg text-sm text-white bg-green-700 bg-gradient-to-r from-green-600 to-green-700 hover:from-green-700 hover:to-green-800">
          Login with OIDC
        </a>
      </div>
    </div>
  </div>

  <Tooltip :result="tooltip.result" :event="tooltip.event"/>
  <Social/>
</template>


<script>
import Social from './components/Social.vue'
import Tooltip from './components/Tooltip.vue';
import {SERVER_URL} from "@/main";
import Loading from "@/components/Loading";

export default {
  name: 'App',
  components: {
    Loading,
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
    logo() {
      return window.config && window.config.logo && window.config.logo !== '{{ .UI.Logo }}' ? window.config.logo : "";
    },
    header() {
      return window.config && window.config.header && window.config.header !== '{{ .UI.Header }}' ? window.config.header : "Health Status";
    },
    link() {
      return window.config && window.config.link && window.config.link !== '{{ .UI.Link }}' ? window.config.link : null;
    },
    buttons() {
      return window.config && window.config.buttons ? window.config.buttons : [];
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