<template>
  <!-- Loading state -->
  <Loading v-if="!retrievedConfig" class="h-64 w-64 px-4" />

  <!-- Main container -->
  <div
    v-else
    :class="[
      config.oidc && !config.authenticated ? 'hidden' : '',
      'container mx-auto p-5 xl:rounded-xl xl:border xl:shadow-xl xl:my-5 xl:pb-5 dark:bg-gray-800 dark:text-gray-200 dark:border-gray-500'
    ]"
    id="global"
  >
    <!-- Header and logo -->
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-3xl lg:text-4xl xl:text-5xl font-light">{{ header }}</h1>
      <a v-if="link" :href="link" target="_blank" class="w-24 h-24">
        <img :src="logo || require('./assets/logo.svg')" alt="Gatus" class="object-contain w-full h-full" />
      </a>
      <div v-else class="w-24 h-24">
        <img src="./assets/logo.svg" alt="Gatus" class="object-contain w-full h-full" />
      </div>
    </div>

    <!-- Optional buttons -->
    <div v-if="buttons.length" class="mb-4 flex space-x-2">
      <a
        v-for="button in buttons"
        :key="button.name"
        :href="button.link"
        target="_blank"
        class="px-2 py-1 text-sm font-medium text-gray-600 hover:text-gray-500 dark:text-gray-300 dark:hover:text-gray-400 hover:underline"
      >
        {{ button.name }}
      </a>
    </div>

    <!-- Router outlet -->
    <router-view @showTooltip="showTooltip" />

    <!-- Endpoint statuses component -->
    <!-- <EndpointStatuses class="mt-6" /> -->

    <!-- Tooltip & social links -->
    <Tooltip :result="tooltip.result" :event="tooltip.event" />
    <Social />
  </div>

  <!-- Login prompt for unauthenticated OIDC users -->
  <div
    v-if="config.oidc && !config.authenticated"
    class="mx-auto max-w-md pt-12 text-center"
  >
    <img src="./assets/logo.svg" alt="Gatus" class="mx-auto mb-4 w-32 h-32 object-contain" />
    <h2 class="text-4xl font-extrabold mb-6 dark:text-gray-200">Gatus</h2>
    <a
      :href="`${SERVER_URL}/oidc/login`"
      class="block max-w-lg mx-auto w-full py-3 px-4 text-sm text-white bg-green-700 border border-green-800 rounded-md shadow-lg hover:bg-green-600"
    >
      Login with OIDC
    </a>
  </div>
</template>

<script>
import Loading from '@/components/Loading.vue';
import Tooltip from '@/components/Tooltip.vue';
import Social from '@/components/Social.vue';
import { SERVER_URL } from '@/main';

export default {
  name: 'App',
  components: {
    Loading,
    Tooltip,
    Social
  },
  data() {
    return {
      retrievedConfig: false,
      config: { oidc: false, authenticated: true },
      tooltip: { result: null, event: null },
      SERVER_URL
    };
  },
  computed: {
    logo() {
      const cfg = window.config || {};
      return cfg.logo && cfg.logo !== '{{ .UI.Logo }}' ? cfg.logo : '';
    },
    header() {
      const cfg = window.config || {};
      return cfg.header && cfg.header !== '{{ .UI.Header }}' ? cfg.header : 'Health Status';
    },
    link() {
      const cfg = window.config || {};
      return cfg.link && cfg.link !== '{{ .UI.Link }}' ? cfg.link : null;
    },
    buttons() {
      const cfg = window.config || {};
      return Array.isArray(cfg.buttons) ? cfg.buttons : [];
    }
  },
  methods: {
    fetchConfig() {
      fetch(`${SERVER_URL}/api/v1/config`, { credentials: 'include' })
        .then(res => {
          this.retrievedConfig = true;
          if (res.ok) res.json().then(data => (this.config = data));
        });
    },
    showTooltip(result, event) {
      this.tooltip = { result, event };
    }
  },
  created() {
    this.fetchConfig();
  }
};
</script>

<!-- All styling is handled via Tailwind utility classes in the template -->
