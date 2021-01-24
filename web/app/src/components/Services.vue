<template>
  <div class="container mx-auto rounded shadow-xl border my-3 p-5 text-left" id="global">
    <div class="mb-2">
      <div class="flex flex-wrap">
        <div class="w-2/3 text-left my-auto">
          <div class="title font-light">Health Status</div>
        </div>
        <div class="w-1/3 flex justify-end">
          <img src="../assets/logo.png" alt="Gatus" style="min-width: 50px; max-width: 200px; width: 20%;"/>
        </div>
      </div>
    </div>
    <div id="results">
      <slot v-for="serviceGroup in serviceGroups" :key="serviceGroup">
        <ServiceGroup :services="serviceGroup.services" :name="serviceGroup.name" />
      </slot>
    </div>
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
    maximumNumberOfResults: Number,
    showStatusOnHover: Boolean,
    serviceStatuses: Object
  },
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
          serviceGroups.push({ name: name, services: outputByGroup[name]})
        }
      }
      // Add all services that don't have a group at the end
      if (outputByGroup['undefined']) {
        serviceGroups.push({name: 'undefined', services: outputByGroup['undefined']})
      }
      this.serviceGroups = serviceGroups;
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
#global {
  max-width: 1140px;
}

#results div.container:first-child {
  border-top-left-radius: 3px;
  border-top-right-radius: 3px;
}

#results div.container:last-child {
  border-bottom-left-radius: 3px;
  border-bottom-right-radius: 3px;
  border-bottom-width: 1px;
  border-color: #dee2e6;
  border-style: solid;
}

#results .service-group-content > div:nth-child(1) {
  border-top-left-radius: 0;
  border-top-right-radius: 0;
}

.title {
  font-size: 2.5rem;
}
</style>
