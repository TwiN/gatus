<template>
  <div class="container container-xs relative mx-auto xl:rounded xl:border xl:shadow-xl xl:my-5 p-5 pb-12 xl:pb-5 text-left" id="global">
    <div class="mb-2">
      <div class="flex flex-wrap">
        <div class="w-3/4 text-left my-auto">
          <div class="text-3xl xl:text-5xl lg:text-4xl font-light">Health Status</div>
        </div>
        <div class="w-1/4 flex justify-end">
          <img src="./assets/logo.png" alt="Gatus" class="object-scale-down" style="max-width: 100px; min-width: 50px; min-height:50px;"/>
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
html {
  height: 100%;
}

html, body {
  background-color: #f7f9fb;
}

#global, #results {
  max-width: 1200px;
}
</style>
