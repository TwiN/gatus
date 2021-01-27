<template>
  <div class="container container-xs relative mx-auto rounded shadow-xl border my-3 p-5 text-left" id="global">
    <div class="mb-2">
      <div class="flex flex-wrap">
        <div class="w-2/3 text-left my-auto">
          <div class="title text-5xl font-light">Health Status</div>
        </div>
        <div class="w-1/3 flex justify-end">
          <img src="./assets/logo.png" alt="Gatus" style="min-width: 50px; max-width: 200px; width: 20%;"/>
        </div>
      </div>
    </div>
    <router-view @showTooltip="showTooltip"/>
  </div>
  <Tooltip :result="tooltip.result" :event="tooltip.event"/>
  <Social/>
</template>


<script>
import Social from './components/Social.vue'
import Tooltip from './components/Tooltip.vue';

export default {
  name: 'App',
  components: {
    Social,
    Tooltip
  },
  methods: {
    showTooltip(result, event) {
      this.tooltip = {result: result, event: event};
    }
  },
  data() {
    return {
      tooltip: {}
    }
  },
}
</script>


<style>
html, body {
  background-color: #f7f9fb;
  height: 100%;
}

#global, #results {
  max-width: 1200px;
}
</style>
