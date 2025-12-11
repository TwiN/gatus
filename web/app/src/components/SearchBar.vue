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
          @input="handleSearchChange($event.target.value, false)"
          @blur="handleSearchChange($event.target.value, true)"
          @keyup.enter="handleSearchChange($event.target.value, true)"
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

const STORAGE_KEYS = {
  FILTER_BY: 'endpointFilterBy',
  SORT_BY: 'endpointSortBy',
}

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
const filterBy = ref(route.query.filter || localStorage.getItem(STORAGE_KEYS.FILTER_BY) || defaultFilterBy)
const sortBy = ref(route.query.sort || localStorage.getItem(STORAGE_KEYS.SORT_BY) || defaultSortBy)

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

const handleSearchChange = (value, push = true) => {
  searchQuery.value = value
  const query = { ...route.query }
  query.search = searchQuery.value || undefined
  push ? router.push({ query }) : router.replace({ query })
  
  emit('search', searchQuery.value)
}

const handleFilterChange = (value, store = true) => {
  filterBy.value = value
  if (store) {
    const query = { ...route.query }
    query.filter = value
    router.push({ query })
    localStorage.setItem(STORAGE_KEYS.FILTER_BY, value)
  }

  emit('update:showOnlyFailing', value === 'failing')
  emit('update:showRecentFailures', value === 'unstable')
}

const handleSortChange = (value, store = true) => {
  sortBy.value = value
  if (store) {
    const query = { ...route.query }
    query.sort = value
    router.push({ query })
    localStorage.setItem(STORAGE_KEYS.SORT_BY, value)
  }

  emit('update:sortBy', value)
  emit('update:groupByGroup', value === 'group')

  // When switching to group view, initialize collapsed groups
  if (value === 'group') {
    emit('initializeCollapsedGroups')
  }
}

onMounted(() => {
  if (route.query.search)
    emit('search', searchQuery.value)

  // Apply saved or application wide filter/sort state on load but do not store it in localstorage
  handleFilterChange(filterBy.value, false)
  handleSortChange(sortBy.value, false)
})

// Ensure browser history navigation (back/forward) re-applies search, filter, and sort
watch(
  () => route.query.search,
  (value) => {
    handleSearchChange(value || '')
  }
)

watch(
  () => route.query.filter,
  (value) => {
    handleFilterChange(value || localStorage.getItem(STORAGE_KEYS.FILTER_BY) || defaultFilterBy, false)
  }
)

watch(
  () => route.query.sort,
  (value) => {
    handleSortChange(value || localStorage.getItem(STORAGE_KEYS.SORT_BY) || defaultSortBy, false)
  }
)
</script>