<template>
  <div :class="endpoints.length === 0 ? 'mt-3' : 'mt-4'">
    <slot v-if="name !== 'undefined'">
      <div class="endpoint-group pt-2 border dark:bg-gray-800 dark:border-gray-500" @click="toggleGroup">
        <h5 class="font-mono text-gray-400 text-xl font-medium pb-2 px-3 dark:text-gray-200 dark:hover:text-gray-500 dark:border-gray-500">
          <span class="endpoint-group-arrow mr-2">
            {{ collapsed ? '&#9660;' : '&#9650;' }}
          </span>
          {{ name }}
          <span v-if="unhealthyCount" class="rounded-xl bg-red-600 text-white px-2 font-bold leading-6 float-right h-6 text-center hover:scale-110 text-sm" title="Partial Outage">{{unhealthyCount}}</span>
          <span v-else class="float-right text-green-600 w-7 hover:scale-110" title="Operational">
            <CheckCircleIcon />
          </span>
        </h5>
      </div>
    </slot>
    <div v-if="!collapsed" :class="name === 'undefined' ? '' : 'endpoint-group-content'">
      <slot v-for="(endpoint, idx) in endpoints" :key="idx">
        <Endpoint
            :data="endpoint"
            :maximumNumberOfResults="20"
            @showTooltip="showTooltip"
            @toggleShowAverageResponseTime="toggleShowAverageResponseTime" :showAverageResponseTime="showAverageResponseTime"
        />
      </slot>
    </div>
  </div>
</template>


<script>
import Endpoint from './Endpoint.vue';
import { CheckCircleIcon } from '@heroicons/vue/20/solid'

export default {
  name: 'EndpointGroup',
  components: {
    Endpoint,
    CheckCircleIcon
  },
  props: {
    name: String,
    endpoints: Array,
    showAverageResponseTime: Boolean
  },
  emits: ['showTooltip', 'toggleShowAverageResponseTime'],
  methods: {
    healthCheck() {
      let unhealthyCount = 0
      if (this.endpoints) {
        for (let i in this.endpoints) {
          if (this.endpoints[i].results && this.endpoints[i].results.length > 0) {
            if (!this.endpoints[i].results[this.endpoints[i].results.length-1].success) {
              unhealthyCount++
            }
          }
        }
      }
      this.unhealthyCount = unhealthyCount;
    },
    toggleGroup() {
      this.collapsed = !this.collapsed;
      localStorage.setItem(`gatus:endpoint-group:${this.name}:collapsed`, this.collapsed);
    },
    showTooltip(result, event) {
      this.$emit('showTooltip', result, event);
    },
    toggleShowAverageResponseTime() {
      this.$emit('toggleShowAverageResponseTime');
    }
  },
  watch: {
    endpoints: function () {
      this.healthCheck();
    }
  },
  created() {
    this.healthCheck();
  },
  data() {
    return {
      unhealthyCount: 0,
      collapsed: localStorage.getItem(`gatus:endpoint-group:${this.name}:collapsed`) === "true"
    }
  }
}
</script>


<style>
.endpoint-group {
  cursor: pointer;
  user-select: none;
}

.endpoint-group h5:hover {
  color: #1b1e21;
}
</style>