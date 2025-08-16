<template>
  <div class="dashboard-container bg-background">
    <div class="container mx-auto px-4 py-8 max-w-7xl">
      <div class="mb-8">
        <div class="flex items-center justify-between mb-6">
          <div>
            <h1 class="text-4xl font-bold tracking-tight">Health Dashboard</h1>
            <p class="text-muted-foreground mt-2">Monitor the health of your endpoints in real-time</p>
          </div>
          <div class="flex items-center gap-4">
            <Button 
              variant="ghost" 
              size="icon" 
              @click="toggleShowAverageResponseTime" 
              :title="showAverageResponseTime ? 'Show min-max response time' : 'Show average response time'"
            >
              <Activity v-if="showAverageResponseTime" class="h-5 w-5" />
              <Timer v-else class="h-5 w-5" />
            </Button>
            <Button variant="ghost" size="icon" @click="refreshData" title="Refresh data">
              <RefreshCw class="h-5 w-5" />
            </Button>
          </div>
        </div>
        
        <SearchBar
          @search="handleSearch"
          @update:showOnlyFailing="showOnlyFailing = $event"
          @update:showRecentFailures="showRecentFailures = $event"
          @update:groupByGroup="groupByGroup = $event"
          @update:sortBy="sortBy = $event"
          @initializeCollapsedGroups="initializeCollapsedGroups"
        />
      </div>

      <!-- Announcements Banner -->
      <AnnouncementBanner :announcements="props.announcements" />

      <div>
      </div>
      <div v-if="loading" class="flex items-center justify-center py-20">
        <Loading size="lg" />
      </div>

      <div v-else-if="filteredEndpoints.length === 0" class="text-center py-20">
        <AlertCircle class="h-12 w-12 text-muted-foreground mx-auto mb-4" />
        <h3 class="text-lg font-semibold mb-2">No endpoints found</h3>
        <p class="text-muted-foreground">
          {{ searchQuery || showOnlyFailing || showRecentFailures 
            ? 'Try adjusting your filters' 
            : 'No endpoints are configured' }}
        </p>
      </div>

      <div v-else>
        <!-- Grouped view -->
        <div v-if="groupByGroup" class="space-y-6">
          <div v-for="(endpoints, group) in paginatedEndpoints" :key="group" class="endpoint-group border rounded-lg overflow-hidden">
            <!-- Group Header -->
            <div 
              @click="toggleGroupCollapse(group)"
              class="endpoint-group-header flex items-center justify-between p-4 bg-card border-b cursor-pointer hover:bg-accent/50 transition-colors"
            >
              <div class="flex items-center gap-3">
                <ChevronDown v-if="!collapsedGroups.has(group)" class="h-5 w-5 text-muted-foreground" />
                <ChevronUp v-else class="h-5 w-5 text-muted-foreground" />
                <h2 class="text-xl font-semibold text-foreground">{{ group }}</h2>
              </div>
              <div class="flex items-center gap-2">
                <span v-if="calculateUnhealthyCount(endpoints) > 0" 
                      class="bg-red-600 text-white px-2 py-1 rounded-full text-sm font-medium">
                  {{ calculateUnhealthyCount(endpoints) }}
                </span>
                <CheckCircle v-else class="h-6 w-6 text-green-600" />
              </div>
            </div>
            
            <!-- Group Content -->
            <div v-if="!collapsedGroups.has(group)" class="endpoint-group-content p-4">
              <div class="grid gap-3 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
                <EndpointCard
                  v-for="endpoint in endpoints"
                  :key="endpoint.key"
                  :endpoint="endpoint"
                  :maxResults="50"
                  :showAverageResponseTime="showAverageResponseTime"
                  @showTooltip="showTooltip"
                />
              </div>
            </div>
          </div>
        </div>
        
        <!-- Regular view -->
        <div v-else class="grid gap-3 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
          <EndpointCard
            v-for="endpoint in paginatedEndpoints"
            :key="endpoint.key"
            :endpoint="endpoint"
            :maxResults="50"
            :showAverageResponseTime="showAverageResponseTime"
            @showTooltip="showTooltip"
          />
        </div>

        <div v-if="!groupByGroup && totalPages > 1" class="mt-8 flex items-center justify-center gap-2">
          <Button
            variant="outline"
            size="icon"
            :disabled="currentPage === 1"
            @click="goToPage(currentPage - 1)"
          >
            <ChevronLeft class="h-4 w-4" />
          </Button>
          
          <div class="flex gap-1">
            <Button
              v-for="page in visiblePages"
              :key="page"
              :variant="page === currentPage ? 'default' : 'outline'"
              size="sm"
              @click="goToPage(page)"
            >
              {{ page }}
            </Button>
          </div>

          <Button
            variant="outline"
            size="icon"
            :disabled="currentPage === totalPages"
            @click="goToPage(currentPage + 1)"
          >
            <ChevronRight class="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>

    <Settings @refreshData="fetchData" />
  </div>
</template>

<script setup>
/* eslint-disable no-undef */
import { ref, computed, onMounted } from 'vue'
import { Activity, Timer, RefreshCw, AlertCircle, ChevronLeft, ChevronRight, ChevronDown, ChevronUp, CheckCircle } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import EndpointCard from '@/components/EndpointCard.vue'
import SearchBar from '@/components/SearchBar.vue'
import Settings from '@/components/Settings.vue'
import Loading from '@/components/Loading.vue'
import AnnouncementBanner from '@/components/AnnouncementBanner.vue'
import { SERVER_URL } from '@/main.js'

const props = defineProps({
  announcements: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['showTooltip'])

const endpointStatuses = ref([])
const loading = ref(false)
const currentPage = ref(1)
const itemsPerPage = 96
const searchQuery = ref('')
const showOnlyFailing = ref(false)
const showRecentFailures = ref(false)
const showAverageResponseTime = ref(true)
const groupByGroup = ref(false)
const sortBy = ref(localStorage.getItem('gatus:sort-by') || 'name')
const collapsedGroups = ref(new Set())

const filteredEndpoints = computed(() => {
  let filtered = [...endpointStatuses.value]
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    filtered = filtered.filter(endpoint => 
      endpoint.name.toLowerCase().includes(query) ||
      (endpoint.group && endpoint.group.toLowerCase().includes(query))
    )
  }
  
  if (showOnlyFailing.value) {
    filtered = filtered.filter(endpoint => {
      if (!endpoint.results || endpoint.results.length === 0) return false
      const latestResult = endpoint.results[endpoint.results.length - 1]
      return !latestResult.success
    })
  }
  
  if (showRecentFailures.value) {
    filtered = filtered.filter(endpoint => {
      if (!endpoint.results || endpoint.results.length === 0) return false
      return endpoint.results.some(result => !result.success)
    })
  }
  
  // Sort by health if selected
  if (sortBy.value === 'health') {
    filtered.sort((a, b) => {
      const aHealthy = a.results && a.results.length > 0 && a.results[a.results.length - 1].success
      const bHealthy = b.results && b.results.length > 0 && b.results[b.results.length - 1].success
      
      // Unhealthy first
      if (!aHealthy && bHealthy) return -1
      if (aHealthy && !bHealthy) return 1
      
      // Then sort by name
      return a.name.localeCompare(b.name)
    })
  }
  
  return filtered
})

const totalPages = computed(() => {
  return Math.ceil(filteredEndpoints.value.length / itemsPerPage)
})

const groupedEndpoints = computed(() => {
  if (!groupByGroup.value) {
    return null
  }
  
  const grouped = {}
  filteredEndpoints.value.forEach(endpoint => {
    const group = endpoint.group || 'No Group'
    if (!grouped[group]) {
      grouped[group] = []
    }
    grouped[group].push(endpoint)
  })
  
  // Sort groups alphabetically, with 'No Group' at the end
  const sortedGroups = Object.keys(grouped).sort((a, b) => {
    if (a === 'No Group') return 1
    if (b === 'No Group') return -1
    return a.localeCompare(b)
  })
  
  const result = {}
  sortedGroups.forEach(group => {
    result[group] = grouped[group]
  })
  
  return result
})

const paginatedEndpoints = computed(() => {
  if (groupByGroup.value) {
    // When grouping, we don't paginate
    return groupedEndpoints.value
  }
  
  const start = (currentPage.value - 1) * itemsPerPage
  const end = start + itemsPerPage
  return filteredEndpoints.value.slice(start, end)
})

const visiblePages = computed(() => {
  const pages = []
  const maxVisible = 5
  let start = Math.max(1, currentPage.value - Math.floor(maxVisible / 2))
  let end = Math.min(totalPages.value, start + maxVisible - 1)
  
  if (end - start < maxVisible - 1) {
    start = Math.max(1, end - maxVisible + 1)
  }
  
  for (let i = start; i <= end; i++) {
    pages.push(i)
  }
  
  return pages
})

const fetchData = async () => {
  loading.value = true
  try {
    const response = await fetch(`${SERVER_URL}/api/v1/endpoints/statuses?page=1&pageSize=100`, {
      credentials: 'include'
    })
    
    if (response.status === 200) {
      const data = await response.json()
      endpointStatuses.value = data
    } else {
      console.error('[Home][fetchData] Error:', await response.text())
    }
  } catch (error) {
    console.error('[Home][fetchData] Error:', error)
  } finally {
    loading.value = false
  }
}

const refreshData = () => {
  fetchData()
}

const handleSearch = (query) => {
  searchQuery.value = query
  currentPage.value = 1
}

const goToPage = (page) => {
  currentPage.value = page
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

const toggleShowAverageResponseTime = () => {
  showAverageResponseTime.value = !showAverageResponseTime.value
}

const showTooltip = (result, event) => {
  emit('showTooltip', result, event)
}

const calculateUnhealthyCount = (endpoints) => {
  return endpoints.filter(endpoint => {
    if (!endpoint.results || endpoint.results.length === 0) return false
    const latestResult = endpoint.results[endpoint.results.length - 1]
    return !latestResult.success
  }).length
}

const toggleGroupCollapse = (groupName) => {
  if (collapsedGroups.value.has(groupName)) {
    collapsedGroups.value.delete(groupName)
  } else {
    collapsedGroups.value.add(groupName)
  }
  // Save to localStorage
  const collapsed = Array.from(collapsedGroups.value)
  localStorage.setItem('gatus:collapsed-groups', JSON.stringify(collapsed))
}

const initializeCollapsedGroups = () => {
  // Get saved collapsed groups from localStorage
  try {
    const saved = localStorage.getItem('gatus:collapsed-groups')
    if (saved) {
      collapsedGroups.value = new Set(JSON.parse(saved))
    }
  } catch (e) {
    console.warn('Failed to parse saved collapsed groups:', e)
    localStorage.removeItem('gatus:collapsed-groups')
  }
}

onMounted(() => {
  fetchData()
})
</script>