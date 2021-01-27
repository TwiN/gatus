<template>
  <div id="results">
    <slot v-for="serviceGroup in serviceGroups" :key="serviceGroup">
      <ServiceGroup :services="serviceGroup.services" :name="serviceGroup.name" @showTooltip="showTooltip"/>
    </slot>
  </div>
</template>


<script>
import ServiceGroup from './ServiceGroup.vue';

export default {
  name: 'Services',
  components: {
    ServiceGroup
  },
  props: {
    showStatusOnHover: Boolean,
    serviceStatuses: Object
  },
  emits: ['showTooltip'],
  methods: {
    process() {
      let outputByGroup = {};
      for (let serviceStatusIndex in this.serviceStatuses) {
        let serviceStatus = this.serviceStatuses[serviceStatusIndex];
        // create an empty entry if this group is new
        if (!outputByGroup[serviceStatus.group] || outputByGroup[serviceStatus.group].length === 0) {
          outputByGroup[serviceStatus.group] = [];
        }
        outputByGroup[serviceStatus.group].push(serviceStatus);
      }
      let serviceGroups = [];
      for (let name in outputByGroup) {
        if (name !== 'undefined') {
          serviceGroups.push({name: name, services: outputByGroup[name]})
        }
      }
      // Add all services that don't have a group at the end
      if (outputByGroup['undefined']) {
        serviceGroups.push({name: 'undefined', services: outputByGroup['undefined']})
      }
      this.serviceGroups = serviceGroups;
    },
    showTooltip(result, event) {
      this.$emit('showTooltip', result, event);
    }
  },
  watch: {
    serviceStatuses: function () {
      this.process();
    }
  },
  data() {
    return {
      userClickedStatus: false,
      serviceGroups: []
    }
  }
}
</script>


<style>
.service-group-content > div:nth-child(1) {
  border-top-left-radius: 0;
  border-top-right-radius: 0;
}
</style>
