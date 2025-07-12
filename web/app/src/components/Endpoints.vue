<template>
  <div id="results">
    <EndpointGroup
      v-for="endpointGroup in endpointGroups"
      :key="endpointGroup.name"
      :endpoints="endpointGroup.endpoints"
      :name="endpointGroup.name"
      @showTooltip="showTooltip"
      @toggleShowAverageResponseTime="toggleShowAverageResponseTime"
      :showAverageResponseTime="showAverageResponseTime"
    />
  </div>
</template>

<script>
import EndpointGroup from './EndpointGroup.vue';

export default {
  name: 'Endpoints',
  components: {
    EndpointGroup
  },
  props: {
    showStatusOnHover: Boolean,
    endpointStatuses: {
      type: Array,
      default: () => []
    },
    showAverageResponseTime: Boolean
  },
  emits: ['showTooltip', 'toggleShowAverageResponseTime'],
  watch: {
    endpointStatuses: {
      immediate: true,
      handler() {
        this.process();
      }
    }
  },
  methods: {
    process() {
      if (!Array.isArray(this.endpointStatuses)) {
        this.endpointGroups = [];
        return;
      }

      const outputByGroup = {};

      for (const endpointStatus of this.endpointStatuses) {
        const groupName = endpointStatus.group || 'Ungrouped';
        if (!outputByGroup[groupName]) {
          outputByGroup[groupName] = [];
        }
        outputByGroup[groupName].push(endpointStatus);
      }

      const endpointGroups = [];

      for (const name in outputByGroup) {
        endpointGroups.push({
          name,
          endpoints: outputByGroup[name]
        });
      }

      this.endpointGroups = endpointGroups;
    },
    showTooltip(result, event) {
      this.$emit('showTooltip', result, event);
    },
    toggleShowAverageResponseTime() {
      this.$emit('toggleShowAverageResponseTime');
    }
  },
  data() {
    return {
      endpointGroups: []
    }
  }
}
</script>

<style>
.endpoint-group-content > div:nth-child(1) {
  border-top-left-radius: 0;
  border-top-right-radius: 0;
}
</style>
