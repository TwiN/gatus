<template>
  <div id="settings" class="flex bg-gray-200 border-gray-300 rounded border shadow dark:text-gray-200 dark:bg-gray-800 dark:border-gray-500">
    <div class="text-xs text-gray-600 rounded-xl py-1.5 px-1.5 dark:text-gray-200">
      <ArrowPathIcon class="w-3"/>
    </div>
    <select class="text-center text-gray-500 text-xs dark:text-gray-200 dark:bg-gray-800 border-r border-l border-gray-300 dark:border-gray-500 pl-1" id="refresh-rate" ref="refreshInterval" @change="handleChangeRefreshInterval">
      <option value="10" :selected="refreshInterval === 10">10s</option>
      <option value="30" :selected="refreshInterval === 30">30s</option>
      <option value="60" :selected="refreshInterval === 60">1m</option>
      <option value="120" :selected="refreshInterval === 120">2m</option>
      <option value="300" :selected="refreshInterval === 300">5m</option>
      <option value="600" :selected="refreshInterval === 600">10m</option>
    </select>
    <button @click="toggleDarkMode" class="text-xs p-1">
      <slot v-if="darkMode"><SunIcon class="w-4"/></slot>
      <slot v-else><MoonIcon class="w-4 text-gray-500"/></slot>
    </button>
  </div>
</template>


<script>
import { MoonIcon, SunIcon } from '@heroicons/vue/20/solid'
import { ArrowPathIcon } from '@heroicons/vue/24/solid'

function wantsDarkMode() {
  const themeFromCookie = document.cookie.match(/theme=(dark|light);?/)?.[1];
  return themeFromCookie === 'dark' || !themeFromCookie && (window.matchMedia('(prefers-color-scheme: dark)').matches || document.documentElement.classList.contains("dark"));
}

export default {
  name: 'Settings',
  components: {
    ArrowPathIcon,
    MoonIcon,
    SunIcon
  },
  props: {},
  methods: {
    setRefreshInterval(seconds) {
      localStorage.setItem('gatus:refresh-interval', seconds);
      let that = this;
      this.refreshIntervalHandler = setInterval(function () {
        that.refreshData();
      }, seconds * 1000);
    },
    refreshData() {
      this.$emit('refreshData');
    },
    handleChangeRefreshInterval() {
      this.refreshData();
      clearInterval(this.refreshIntervalHandler);
      this.setRefreshInterval(this.$refs.refreshInterval.value);
    },
    toggleDarkMode() {
      if (wantsDarkMode()) {
        document.cookie = `theme=light; path=/; max-age=31536000; samesite=strict`;
      } else {
        document.cookie = `theme=dark; path=/; max-age=31536000; samesite=strict`;
      }
      this.applyTheme();
    },
    applyTheme() {
      if (wantsDarkMode()) {
        this.darkMode = true;
        document.documentElement.classList.add('dark');
      } else {
        this.darkMode = false;
        document.documentElement.classList.remove('dark');
      }
    }
  },
  created() {
    if (this.refreshInterval !== 10 && this.refreshInterval !== 30 && this.refreshInterval !== 60 && this.refreshInterval !== 120 && this.refreshInterval !== 300 && this.refreshInterval !== 600) {
      this.refreshInterval = 300;
    }
    this.setRefreshInterval(this.refreshInterval);
    this.applyTheme();
  },
  unmounted() {
    clearInterval(this.refreshIntervalHandler);
  },
  data() {
    return {
      refreshInterval: localStorage.getItem('gatus:refresh-interval') < 10 ? 300 : parseInt(localStorage.getItem('gatus:refresh-interval')),
      refreshIntervalHandler: 0,
      darkMode: wantsDarkMode()
    }
  },
}
</script>


<style>
#settings {
  position: fixed;
  left: 10px;
  bottom: 10px;
}

#settings select:focus {
  box-shadow: none;
}
</style>
