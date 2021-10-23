<template>
  <div id="results">
    <slot v-for="endpointGroup in endpointGroups" :key="endpointGroup">
      <EndpointGroup :endpoints="endpointGroup.endpoints" :name="endpointGroup.name" @showTooltip="showTooltip" @toggleShowAverageResponseTime="toggleShowAverageResponseTime" :showAverageResponseTime="showAverageResponseTime" />
    </slot>
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
    endpointStatuses: Object,
    showAverageResponseTime: Boolean
  },
  emits: ['showTooltip', 'toggleShowAverageResponseTime'],
  methods: {
    process() {
      let outputByGroup = {};
      for (let endpointStatusIndex in this.endpointStatuses) {
        let endpointStatus = this.endpointStatuses[endpointStatusIndex];
        // create an empty entry if this group is new
        if (!outputByGroup[endpointStatus.group] || outputByGroup[endpointStatus.group].length === 0) {
          outputByGroup[endpointStatus.group] = [];
        }
        outputByGroup[endpointStatus.group].push(endpointStatus);
      }
      let endpointGroups = [];
      for (let name in outputByGroup) {
        if (name !== 'undefined') {
          endpointGroups.push({name: name, endpoints: outputByGroup[name]})
        }
      }
      // Add all endpoints that don't have a group at the end
      if (outputByGroup['undefined']) {
        endpointGroups.push({name: 'undefined', endpoints: outputByGroup['undefined']})
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
  watch: {
    endpointStatuses: function () {
      this.process();
    }
  },
  data() {
    return {
      userClickedStatus: false,
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
