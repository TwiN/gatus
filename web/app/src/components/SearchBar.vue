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
          @input="handleSearchChange"
          @blur="normalizeSearchInput"
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
import { useRoute, useRouter } from 'vue-router'
import { Search } from 'lucide-vue-next'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import {
  DASHBOARD_FILTER_VALUES,
  DASHBOARD_SORT_VALUES,
  getFirstQueryValue,
  normalizeDashboardOption,
  normalizeSearchQuery,
  withQueryValue,
} from '@/utils/dashboard-query.mjs'

const router = useRouter()
const route = useRoute()

const storedFilterBy = localStorage.getItem('gatus:filter-by')
const configuredFilterBy = typeof window !== 'undefined' && window.config?.defaultFilterBy
const configuredFilterFallback = normalizeDashboardOption(
  configuredFilterBy,
  DASHBOARD_FILTER_VALUES,
  'none'
)
const filterFallback = normalizeDashboardOption(
  storedFilterBy,
  DASHBOARD_FILTER_VALUES,
  configuredFilterFallback
)

const storedSortBy = localStorage.getItem('gatus:sort-by')
const configuredSortBy = typeof window !== 'undefined' && window.config?.defaultSortBy
const configuredSortFallback = normalizeDashboardOption(
  configuredSortBy,
  DASHBOARD_SORT_VALUES,
  'name'
)
const sortFallback = normalizeDashboardOption(
  storedSortBy,
  DASHBOARD_SORT_VALUES,
  configuredSortFallback
)

const searchQuery = ref(normalizeSearchQuery(route.query.search))
const filterBy = ref(normalizeDashboardOption(route.query.filter, DASHBOARD_FILTER_VALUES, filterFallback))
const sortBy = ref(normalizeDashboardOption(route.query.sort, DASHBOARD_SORT_VALUES, sortFallback))

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

const handleSearchChange = event => {
  searchQuery.value = event.target.value
  const normalizedValue = normalizeSearchQuery(searchQuery.value)
  router.replace({ query: withQueryValue(route.query, 'search', normalizedValue) })
  emit('search', normalizedValue)
}

const normalizeSearchInput = () => {
  const normalizedValue = normalizeSearchQuery(searchQuery.value)
  if (normalizedValue !== searchQuery.value) {
    searchQuery.value = normalizedValue
    router.replace({ query: withQueryValue(route.query, 'search', normalizedValue) })
  }
}

const handleFilterChange = (value, store = true) => {
  filterBy.value = value
  if (store) {
    localStorage.setItem('gatus:filter-by', value)
    router.push({ query: withQueryValue(route.query, 'filter', value) })
  }

  emit('update:showOnlyFailing', value === 'failing')
  emit('update:showRecentFailures', value === 'unstable')
}

const handleSortChange = (value, store = true) => {
  sortBy.value = value
  if (store) {
    localStorage.setItem('gatus:sort-by', value)
    router.push({ query: withQueryValue(route.query, 'sort', value) })
  }

  emit('update:sortBy', value)
  emit('update:groupByGroup', value === 'group')
  
  // When switching to group view, initialize collapsed groups
  if (value === 'group') {
    emit('initializeCollapsedGroups')
  }
}

onMounted(() => {
  emit('search', searchQuery.value)

  // Canonicalize URL-provided search values once without disrupting multi-word typing.
  const routeSearchQuery = getFirstQueryValue(route.query.search)
  if (
    route.query.search !== undefined &&
    (Array.isArray(route.query.search) || routeSearchQuery !== searchQuery.value || searchQuery.value === '')
  ) {
    router.replace({ query: withQueryValue(route.query, 'search', searchQuery.value) })
  }

  // Apply saved or application wide filter/sort state on load but do not store it in localstorage
  handleFilterChange(filterBy.value, false)
  handleSortChange(sortBy.value, false)
})

watch(
  () => route.query.search,
  value => {
    const normalizedValue = normalizeSearchQuery(value)
    // Compare normalized values so URL updates do not remove a trailing space
    // while the user is in the middle of entering a multi-word search.
    if (normalizedValue !== normalizeSearchQuery(searchQuery.value)) {
      searchQuery.value = normalizedValue
      emit('search', normalizedValue)
    }
  }
)

watch(
  () => route.query.filter,
  value => {
    const normalizedValue = normalizeDashboardOption(value, DASHBOARD_FILTER_VALUES, filterFallback)
    if (normalizedValue !== filterBy.value) {
      // Apply route-driven state without writing the same value back to the URL.
      handleFilterChange(normalizedValue, false)
    }
  }
)

watch(
  () => route.query.sort,
  value => {
    const normalizedValue = normalizeDashboardOption(value, DASHBOARD_SORT_VALUES, sortFallback)
    if (normalizedValue !== sortBy.value) {
      handleSortChange(normalizedValue, false)
    }
  }
)
</script>
