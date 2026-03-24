<template>
  <div class="dashboard-container bg-background">
    <div class="container mx-auto px-4 py-8 max-w-7xl">
      <div class="mb-6">
        <Button variant="ghost" class="mb-4" @click="goBack">
          <ArrowLeft class="h-4 w-4 mr-2" />
          Back to Dashboard
        </Button>
        
        <div v-if="endpointStatus && endpointStatus.name" class="space-y-6">
          <div class="flex items-start justify-between">
            <div>
              <h1 class="text-4xl font-bold tracking-tight">{{ endpointStatus.name }}</h1>
              <div class="flex items-center gap-3 text-muted-foreground mt-2">
                <span v-if="endpointStatus.group">Group: {{ endpointStatus.group }}</span>
                <span v-if="endpointStatus.group && hostname">•</span>
                <span v-if="hostname">{{ hostname }}</span>
              </div>
            </div>
            <StatusBadge :status="currentHealthStatus" />
          </div>

          <div class="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
            <Card>
              <CardHeader class="pb-2">
                <CardTitle class="text-sm font-medium text-muted-foreground">Current Status</CardTitle>
              </CardHeader>
              <CardContent>
                <div class="text-2xl font-bold">{{ currentHealthStatus === 'healthy' ? 'Operational' : 'Issues Detected' }}</div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader class="pb-2">
                <CardTitle class="text-sm font-medium text-muted-foreground">Avg Response Time</CardTitle>
              </CardHeader>
              <CardContent>
                <div class="text-2xl font-bold">{{ pageAverageResponseTime }}</div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader class="pb-2">
                <CardTitle class="text-sm font-medium text-muted-foreground">Response Time Range</CardTitle>
              </CardHeader>
              <CardContent>
                <div class="text-2xl font-bold">{{ pageResponseTimeRange }}</div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader class="pb-2">
                <CardTitle class="text-sm font-medium text-muted-foreground">Last Check</CardTitle>
              </CardHeader>
              <CardContent>
                <div class="text-2xl font-bold">{{ lastCheckTime }}</div>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <div class="flex items-center justify-between">
                <CardTitle>Recent Checks</CardTitle>
                <div class="flex items-center gap-2">
                  <Button 
                    variant="ghost" 
                    size="icon"
                    @click="toggleShowAverageResponseTime"
                    :title="showAverageResponseTime ? 'Show min-max response time' : 'Show average response time'"
                  >
                    <Activity v-if="showAverageResponseTime" class="h-5 w-5" />
                    <Timer v-else class="h-5 w-5" />
                  </Button>
                  <Button 
                    variant="ghost" 
                    size="icon" 
                    @click="fetchData"
                    title="Refresh data"
                    :disabled="isRefreshing"
                  >
                    <RefreshCw :class="['h-4 w-4', isRefreshing && 'animate-spin']" />
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div class="space-y-4">
                <EndpointCard 
                  v-if="endpointStatus"
                  :endpoint="endpointStatus"
                  :maxResults="resultPageSize"
                  :showAverageResponseTime="showAverageResponseTime"
                  @showTooltip="showTooltip"
                  class="border-0 shadow-none bg-transparent p-0"
                />
                <div v-if="endpointStatus && endpointStatus.key" class="pt-4 border-t">
                  <Pagination @page="changePage" :numberOfResultsPerPage="resultPageSize" :currentPageProp="currentPage" />
                </div>
              </div>
            </CardContent>
          </Card>

          <div v-if="showResponseTimeChartAndBadges" class="space-y-6">
            <Card>
              <CardHeader>
                <div class="flex items-center justify-between">
                  <CardTitle>Response Time Trend</CardTitle>
                  <select 
                    v-model="selectedChartDuration"
                    class="text-sm bg-background border rounded-md px-3 py-1 focus:outline-none focus:ring-2 focus:ring-ring"
                  >
                    <option value="24h">24 hours</option>
                    <option value="7d">7 days</option>
                    <option value="30d">30 days</option>
                  </select>
                </div>
              </CardHeader>
              <CardContent>
                <ResponseTimeChart
                  v-if="endpointStatus && endpointStatus.key"
                  :endpointKey="endpointStatus.key"
                  :duration="selectedChartDuration"
                  :serverUrl="serverUrl"
                  :events="endpointStatus.events || []"
                />
              </CardContent>
            </Card>

            <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              <Card v-for="period in ['30d', '7d', '24h', '1h']" :key="period">
                <CardHeader class="pb-2">
                  <CardTitle class="text-sm font-medium text-muted-foreground text-center">
                    {{ period === '30d' ? 'Last 30 days' : period === '7d' ? 'Last 7 days' : period === '24h' ? 'Last 24 hours' : 'Last hour' }}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <img :src="generateResponseTimeBadgeImageURL(period)" :alt="`${period} response time`" class="mx-auto mt-2" />
                </CardContent>
              </Card>
            </div>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Uptime Statistics</CardTitle>
            </CardHeader>
            <CardContent>
              <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                <div v-for="period in ['30d', '7d', '24h', '1h']" :key="period" class="text-center">
                  <p class="text-sm text-muted-foreground mb-2">
                    {{ period === '30d' ? 'Last 30 days' : period === '7d' ? 'Last 7 days' : period === '24h' ? 'Last 24 hours' : 'Last hour' }}
                  </p>
                  <img :src="generateUptimeBadgeImageURL(period)" :alt="`${period} uptime`" class="mx-auto" />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Current Health</CardTitle>
            </CardHeader>
            <CardContent>
              <div class="text-center">
                <img :src="generateHealthBadgeImageURL()" alt="health badge" class="mx-auto" />
              </div>
            </CardContent>
          </Card>

          <Card v-if="events && events.length > 0">
            <CardHeader>
              <CardTitle>Events</CardTitle>
            </CardHeader>
            <CardContent>
              <div class="space-y-4">
                <div v-for="event in events" :key="event.timestamp" class="flex items-start gap-4 pb-4 border-b last:border-0">
                  <div class="mt-1">
                    <ArrowUpCircle v-if="event.type === 'HEALTHY'" class="h-5 w-5 text-green-500" />
                    <ArrowDownCircle v-else-if="event.type === 'UNHEALTHY'" class="h-5 w-5 text-red-500" />
                    <PlayCircle v-else class="h-5 w-5 text-muted-foreground" />
                  </div>
                  <div class="flex-1">
                    <p class="font-medium">{{ event.fancyText }}</p>
                    <p class="text-sm text-muted-foreground">{{ prettifyTimestamp(event.timestamp) }} • {{ event.fancyTimeAgo }}</p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        <div v-else class="flex items-center justify-center py-20">
          <Loading size="lg" />
        </div>
      </div>
    </div>

    <Settings @refreshData="fetchData" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ArrowLeft, RefreshCw, ArrowUpCircle, ArrowDownCircle, PlayCircle, Activity, Timer } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import StatusBadge from '@/components/StatusBadge.vue'
import EndpointCard from '@/components/EndpointCard.vue'
import Settings from '@/components/Settings.vue'
import Pagination from '@/components/Pagination.vue'
import Loading from '@/components/Loading.vue'
import ResponseTimeChart from '@/components/ResponseTimeChart.vue'
import { generatePrettyTimeAgo, generatePrettyTimeDifference } from '@/utils/time'

const router = useRouter()
const route = useRoute()
const emit = defineEmits(['showTooltip'])

const endpointStatus = ref(null) // For paginated historical data
const currentStatus = ref(null) // For current/latest status (always page 1)
const events = ref([])
const currentPage = ref(1)
const resultPageSize = 50
const showResponseTimeChartAndBadges = ref(false)
const showAverageResponseTime = ref(localStorage.getItem('gatus:show-average-response-time') !== 'false')
const selectedChartDuration = ref('24h')
const isRefreshing = ref(false)

const latestResult = computed(() => {
  // Use currentStatus for the actual latest result
  if (!currentStatus.value || !currentStatus.value.results || currentStatus.value.results.length === 0) {
    return null
  }
  return currentStatus.value.results[currentStatus.value.results.length - 1]
})

const currentHealthStatus = computed(() => {
  if (!latestResult.value) return 'unknown'
  return latestResult.value.success ? 'healthy' : 'unhealthy'
})

const hostname = computed(() => {
  return latestResult.value?.hostname || null
})

const toggleShowAverageResponseTime = () => {
  showAverageResponseTime.value = !showAverageResponseTime.value
  localStorage.setItem('gatus:show-average-response-time', showAverageResponseTime.value ? 'true' : 'false')
}

const pageAverageResponseTime = computed(() => {
  // Use endpointStatus for current page's average response time
  if (!endpointStatus.value || !endpointStatus.value.results || endpointStatus.value.results.length === 0) {
    return 'N/A'
  }
  let total = 0
  let count = 0
  for (const result of endpointStatus.value.results) {
    if (result.duration) {
      total += result.duration
      count++
    }
  }
  if (count === 0) return 'N/A'
  return `${Math.round(total / count / 1000000)}ms`
})

const pageResponseTimeRange = computed(() => {
  // Use endpointStatus for current page's response time range
  if (!endpointStatus.value || !endpointStatus.value.results || endpointStatus.value.results.length === 0) {
    return 'N/A'
  }
  let min = Infinity
  let max = 0
  let hasData = false
  
  for (const result of endpointStatus.value.results) {
    const duration = result.duration
    if (duration) {
      min = Math.min(min, duration)
      max = Math.max(max, duration)
      hasData = true
    }
  }
  
  if (!hasData) return 'N/A'
  const minMs = Math.trunc(min / 1000000)
  const maxMs = Math.trunc(max / 1000000)
  // If min and max are the same, show single value
  if (minMs === maxMs) {
    return `${minMs}ms`
  }
  return `${minMs}-${maxMs}ms`
})

const lastCheckTime = computed(() => {
  // Use currentStatus for real-time last check time
  if (!currentStatus.value || !currentStatus.value.results || currentStatus.value.results.length === 0) {
    return 'Never'
  }
  return generatePrettyTimeAgo(currentStatus.value.results[currentStatus.value.results.length - 1].timestamp)
})


const fetchData = async () => {
  isRefreshing.value = true
  try {
    const response = await fetch(`/api/v1/endpoints/${route.params.key}/statuses?page=${currentPage.value}&pageSize=${resultPageSize}`, {
      credentials: 'include'
    })
    
    if (response.status === 200) {
      const data = await response.json()
      endpointStatus.value = data
      
      // Always update currentStatus when on page 1 (including when returning to it)
      if (currentPage.value === 1) {
        currentStatus.value = data
      }
      
      let processedEvents = []
      if (data.events && data.events.length > 0) {
        for (let i = data.events.length - 1; i >= 0; i--) {
          let event = data.events[i]
          if (i === data.events.length - 1) {
            if (event.type === 'UNHEALTHY') {
              event.fancyText = 'Endpoint is unhealthy'
            } else if (event.type === 'HEALTHY') {
              event.fancyText = 'Endpoint is healthy'
            } else if (event.type === 'START') {
              event.fancyText = 'Monitoring started'
            }
          } else {
            let nextEvent = data.events[i + 1]
            if (event.type === 'HEALTHY') {
              event.fancyText = 'Endpoint became healthy'
            } else if (event.type === 'UNHEALTHY') {
              if (nextEvent) {
                event.fancyText = 'Endpoint was unhealthy for ' + generatePrettyTimeDifference(nextEvent.timestamp, event.timestamp)
              } else {
                event.fancyText = 'Endpoint became unhealthy'
              }
            } else if (event.type === 'START') {
              event.fancyText = 'Monitoring started'
            }
          }
          event.fancyTimeAgo = generatePrettyTimeAgo(event.timestamp)
          processedEvents.push(event)
        }
      }
      events.value = processedEvents
      
      if (data.results && data.results.length > 0) {
        for (let i = 0; i < data.results.length; i++) {
          if (data.results[i].duration > 0) {
            showResponseTimeChartAndBadges.value = true
            break
          }
        }
      }
    } else {
      console.error('[Details][fetchData] Error:', await response.text())
    }
  } catch (error) {
    console.error('[Details][fetchData] Error:', error)
  } finally {
    isRefreshing.value = false
  }
}

const goBack = () => {
  router.push('/')
}

const changePage = (page) => {
  currentPage.value = page
  fetchData()
}

const showTooltip = (result, event, action = 'hover') => {
  emit('showTooltip', result, event, action)
}

const prettifyTimestamp = (timestamp) => {
  return new Date(timestamp).toLocaleString()
}

const generateHealthBadgeImageURL = () => {
  return `/api/v1/endpoints/${endpointStatus.value.key}/health/badge.svg`
}

const generateUptimeBadgeImageURL = (duration) => {
  return `/api/v1/endpoints/${endpointStatus.value.key}/uptimes/${duration}/badge.svg`
}

const generateResponseTimeBadgeImageURL = (duration) => {
  return `/api/v1/endpoints/${endpointStatus.value.key}/response-times/${duration}/badge.svg`
}

onMounted(() => {
  fetchData()
})
</script>