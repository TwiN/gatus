<template>
  <div id="settings">
    <div class="flex bg-gray-200 rounded border border-gray-300 shadow">
      <div class="text-sm text-gray-600 rounded-xl py-1 px-2">
        &#x21bb;
      </div>
      <select class="text-center text-gray-500 text-sm" id="refresh-rate" ref="refreshInterval" @change="handleChangeRefreshInterval">
        <option value="10">10s</option>
        <option value="30" selected>30s</option>
        <option value="60">1m</option>
        <option value="120">2m</option>
        <option value="300">5m</option>
        <option value="600">10m</option>
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
    this.setRefreshInterval(this.refreshInterval);
  },
  unmounted() {
    clearInterval(this.refreshIntervalHandler);
  },
  data() {
    return {
      refreshInterval: 30,
      refreshIntervalHandler: 0,
    }
  },
}

// props.refreshInterval = 30
//$("#refresh-rate").val(30);
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
