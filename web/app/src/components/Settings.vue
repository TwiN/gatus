<template>
  <div id="settings">
    <div class="flex bg-gray-200 rounded border border-gray-300 shadow">
      <div class="text-sm text-gray-600 rounded-xl py-1 px-2">
        &#x21bb;
      </div>
      <select class="text-center text-gray-500 text-sm" id="refresh-rate" ref="refreshInterval" @change="handleChangeRefreshInterval">
        <option value="10" :selected="refreshInterval === 10">10s</option>
        <option value="30" :selected="refreshInterval === 30">30s</option>
        <option value="60" :selected="refreshInterval === 60">1m</option>
        <option value="120" :selected="refreshInterval === 120">2m</option>
        <option value="300" :selected="refreshInterval === 300">5m</option>
        <option value="600" :selected="refreshInterval === 600">10m</option>
      </select>
    </div>
  </div>
</template>


<script>
export default {
  name: 'Settings',
  props: {},
  methods: {
    setRefreshInterval(seconds) {
      sessionStorage.setItem('gatus:refresh-interval', seconds);
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
    }
  },
  created() {
    if (this.refreshInterval !== 10 && this.refreshInterval !== 30 && this.refreshInterval !== 60 && this.refreshInterval !== 120 && this.refreshInterval !== 300 && this.refreshInterval !== 600) {
      this.refreshInterval = 60;
    }
    this.setRefreshInterval(this.refreshInterval);
  },
  unmounted() {
    clearInterval(this.refreshIntervalHandler);
  },
  data() {
    return {
      refreshInterval: sessionStorage.getItem('gatus:refresh-interval') < 10 ? 60 : parseInt(sessionStorage.getItem('gatus:refresh-interval')),
      refreshIntervalHandler: 0,
    }
  },
}
</script>


<style scoped>
#settings {
  position: fixed;
  left: 5px;
  bottom: 5px;
  padding: 5px;
}

#settings select:focus {
  box-shadow: none;
}
</style>
