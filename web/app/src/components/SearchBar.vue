<template>
  <div class="flex flex-col lg:flex-row gap-3 lg:gap-4 p-3 sm:p-4 bg-card rounded-lg border">
    <div class="flex-1">
      <div class="relative">
        <Search class="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <label for="search-input" class="sr-only">Search endpoints</label>
        <Input
          id="search-input"
          v-model="searchQuery"
          type="text"
          placeholder="Search endpoints..."
          class="pl-10 text-sm sm:text-base"
          @input="$emit('search', searchQuery)"
        />
      </div>
    </div>
    <div class="flex flex-col sm:flex-row gap-3 sm:gap-4">
      <div class="flex items-center gap-2 flex-1 sm:flex-initial">
        <label class="text-xs sm:text-sm font-medium text-muted-foreground whitespace-nowrap">Filter by:</label>
        <Select 
          v-model="filterBy" 
          :options="filterOptions"
          placeholder="None"
          class="flex-1 sm:w-[140px] md:w-[160px]"
          @update:model-value="handleFilterChange"
        />
      </div>
      
      <div class="flex items-center gap-2 flex-1 sm:flex-initial">
        <label class="text-xs sm:text-sm font-medium text-muted-foreground whitespace-nowrap">Sort by:</label>
        <Select 
          v-model="sortBy" 
          :options="sortOptions"
          placeholder="Name"
          class="flex-1 sm:w-[90px] md:w-[100px]"
          @update:model-value="handleSortChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Search } from 'lucide-vue-next'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'

const router = useRouter()
const route = useRoute()

const [defaultFilterBy, defaultSortBy] = (() => {
  let filter = 'none'
  let sort = 'name'
  if (typeof window !== 'undefined' && window.config) {
    if (window.config.defaultFilterBy && window.config.defaultFilterBy !== '{{ .UI.DefaultFilterBy }}') {
      filter = window.config.defaultFilterBy
    }
    if (window.config.defaultSortBy && window.config.defaultSortBy !== '{{ .UI.DefaultSortBy }}') {
      sort = window.config.defaultSortBy
    }
  }
  return [filter, sort]
})()

const searchQuery = ref(route.query.search || '')
const filterBy = ref(route.query.filter || localStorage.getItem('gatus:filter-by') || defaultFilterBy)
const sortBy = ref(route.query.sort || localStorage.getItem('gatus:sort-by') || defaultSortBy)

const filterOptions = [
  { label: 'None', value: 'none' },
  { label: 'Failing', value: 'failing' },
  { label: 'Unstable', value: 'unstable' }
]

const sortOptions = [
  { label: 'Name', value: 'name' },
  { label: 'Group', value: 'group' },
  { label: 'Health', value: 'health' }
]

const emit = defineEmits(['search', 'update:showOnlyFailing', 'update:showRecentFailures', 'update:groupByGroup', 'update:sortBy', 'initializeCollapsedGroups'])

const handleFilterChange = (value, store = true) => {
  filterBy.value = value
  router.push({
    query: {
      ...route.query,
      filter: value
    }
  });
  if (store)
    localStorage.setItem('gatus:filter-by', value)
  
  emit('update:showOnlyFailing', value === 'failing')
  emit('update:showRecentFailures', value === 'unstable')
}

const handleSortChange = (value, store = true) => {
  sortBy.value = value
  router.push({
    query: {
      ...route.query,
      sort: value
    }
  });
  if (store)
    localStorage.setItem('gatus:sort-by', value)

  emit('update:sortBy', value)
  emit('update:groupByGroup', value === 'group')

  // When switching to group view, initialize collapsed groups
  if (value === 'group') {
    emit('initializeCollapsedGroups')
  }
}

onMounted(() => {
  // Apply query search state on load
  if (searchQuery.value !== '')
    emit ('search', searchQuery.value)

  // Apply saved or application wide filter/sort state on load but do not store it in localstorage
  handleFilterChange(filterBy.value, false)
  handleSortChange(sortBy.value, false)
})

watch(
  () => route.query.search, (newValue) => {
    searchQuery.value = newValue || ''
  },
  () => route.query.filter, (newValue) => {
    if (!newValue)
      newValue = localStorage.getItem('gatus:filter-by') || defaultFilterBy
    if (newValue !== filterBy.value)
      handleFilterChange(newValue, false)
  },
  () => route.query.sort, (newValue) => {
    if (!newValue)
      newValue = localStorage.getItem('gatus:sort-by') || defaultSortBy
    if (newValue !== sortBy.value)
      handleSortChange(newValue, false)
  }
)

</script>