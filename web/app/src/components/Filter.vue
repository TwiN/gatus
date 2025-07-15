<template>
  <div id="filters" class="flex flex-wrap gap-2 mb-4 p-4 bg-gray-100 border-gray-300 rounded border shadow dark:bg-gray-800 dark:border-gray-500">
    <div class="flex items-center gap-2">
      <label class="text-sm font-medium text-gray-700 dark:text-gray-200">
        Group:
      </label>
      <select 
        v-model="selectedGroup" 
        @change="onFilterChange"
        class="text-sm border border-gray-300 rounded px-2 py-1 bg-white dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
      >
        <option value="">All Groups</option>
        <option v-for="group in availableGroups" :key="group" :value="group">
          {{ group }}
        </option>
      </select>
    </div>
    
    <div class="flex items-center gap-2">
      <label class="text-sm font-medium text-gray-700 dark:text-gray-200">
        Status:
      </label>
      <select 
        v-model="selectedStatus" 
        @change="onFilterChange"
        class="text-sm border border-gray-300 rounded px-2 py-1 bg-white dark:bg-gray-700 dark:border-gray-600 dark:text-gray-200"
      >
        <option value="">All Statuses</option>
        <option value="up">Up</option>
        <option value="down">Down</option>
      </select>
    </div>
    
    <button 
      @click="clearFilters"
      class="text-sm px-3 py-1 bg-gray-500 text-white rounded hover:bg-gray-600 dark:bg-gray-600 dark:hover:bg-gray-700"
    >
      Clear Filters
    </button>
  </div>
</template>

<script>
export default {
  name: 'Filter',
  props: {
    endpointStatuses: Array,
    initialGroup: String,
    initialStatus: String
  },
  emits: ['filterChange'],
  data() {
    return {
      selectedGroup: this.initialGroup || '',
      selectedStatus: this.initialStatus || ''
    }
  },
  computed: {
    availableGroups() {
      if (!this.endpointStatuses) return [];
      
      const groups = new Set();
      this.endpointStatuses.forEach(endpoint => {
        if (endpoint.group && endpoint.group !== 'undefined') {
          groups.add(endpoint.group);
        }
      });
      return Array.from(groups).sort();
    }
  },
  methods: {
    onFilterChange() {
      this.$emit('filterChange', {
        group: this.selectedGroup,
        status: this.selectedStatus
      });
    },
    clearFilters() {
      this.selectedGroup = '';
      this.selectedStatus = '';
      this.onFilterChange();
    }
  },
  watch: {
    initialGroup(newVal) {
      this.selectedGroup = newVal || '';
    },
    initialStatus(newVal) {
      this.selectedStatus = newVal || '';
    }
  }
}
</script>

<style scoped>
select:focus {
  outline: none;
  box-shadow: 0 0 0 2px rgb(59 130 246);
}

button:focus {
  outline: none;
  box-shadow: 0 0 0 2px rgb(59 130 246);
}
</style>
