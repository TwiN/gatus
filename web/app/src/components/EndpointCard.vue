<template>
  <Card class="endpoint h-full flex flex-col transition hover:shadow-lg hover:scale-[1.01] dark:hover:border-gray-700">
    <CardHeader class="endpoint-header px-3 sm:px-6 pt-3 sm:pt-6 pb-2 space-y-0">
      <div class="flex items-start justify-between gap-2 sm:gap-3">
        <div class="flex-1 min-w-0 overflow-hidden">
          <CardTitle class="text-base sm:text-lg truncate">
            <span 
              class="hover:text-primary cursor-pointer hover:underline text-sm sm:text-base block truncate" 
              @click="navigateToDetails" 
              @keydown.enter="navigateToDetails"
              :title="endpoint.name"
              role="link"
              tabindex="0"
              :aria-label="`View details for ${endpoint.name}`">
              {{ endpoint.name }}
            </span>
          </CardTitle>
          <div class="flex items-center gap-2 text-xs sm:text-sm text-muted-foreground min-h-[1.25rem]">
            <span v-if="endpoint.group" class="truncate" :title="endpoint.group">{{ endpoint.group }}</span>
            <span v-if="endpoint.group && hostname">•</span>
            <span v-if="hostname" class="truncate" :title="hostname">{{ hostname }}</span>
          </div>
        </div>
        <div class="flex-shrink-0 ml-2">
          <StatusBadge :status="currentStatus" />
        </div>
      </div>
    </CardHeader>
    <CardContent class="endpoint-content flex-1 pb-3 sm:pb-4 px-3 sm:px-6 pt-2">
      <div class="space-y-2">
        <div>
          <div class="flex items-center justify-between mb-1">
            <div class="flex-1"></div>
            <p class="text-xs text-muted-foreground" :title="showAverageResponseTime ? 'Average response time' : 'Minimum and maximum response time'">{{ formattedResponseTime }}</p>
          </div>
          <div v-if="configuredPeriod && periodLoading" class="flex gap-0.5">
            <div v-for="i in maxResults" :key="i" class="flex-1 h-6 sm:h-8 rounded-sm bg-gray-200 dark:bg-gray-700 animate-pulse" />
          </div>
          <div v-else class="flex gap-0.5">
            <div
              v-for="(item, index) in displayBars"
              :key="index"
              :class="[
                'flex-1 h-6 sm:h-8 rounded-sm transition-all',
                barClickable(item) ? 'cursor-pointer' : '',
                barClass(item, index)
              ]"
              @mouseenter="item && handleMouseEnter(item, $event)"
              @mouseleave="item && handleMouseLeave(item, $event)"
              @click.stop="item && handleClick(item, $event, index)"
            />
          </div>
          <div class="flex items-center gap-1 text-xs text-muted-foreground mt-1">
            <span>{{ displayStartTime }}</span>
            <span class="flex-1 border-t border-dashed border-muted-foreground/30 mx-1"></span>
            <span class="font-medium" :class="uptimeColor(displayUptime)">{{ formatUptimePercent(displayUptime) }} uptime</span>
            <span class="flex-1 border-t border-dashed border-muted-foreground/30 mx-1"></span>
            <span>{{ displayEndTime }}</span>
          </div>
        </div>
      </div>
    </CardContent>
  </Card>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import StatusBadge from '@/components/StatusBadge.vue'
import { generatePrettyTimeAgo } from '@/utils/time'

const router = useRouter()

const props = defineProps({
  endpoint: {
    type: Object,
    required: true
  },
  maxResults: {
    type: Number,
    default: 50
  },
  showAverageResponseTime: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['showTooltip'])

const selectedResultIndex = ref(null)
const periodData = ref(null)
const periodLoading = ref(false)

const configuredPeriod = computed(() => props.endpoint?.period || null)

const latestResult = computed(() => {
  if (!props.endpoint.results || props.endpoint.results.length === 0) {
    return null
  }
  return props.endpoint.results[props.endpoint.results.length - 1]
})

const currentStatus = computed(() => {
  if (!latestResult.value) return 'unknown'
  return latestResult.value.success ? 'healthy' : 'unhealthy'
})

const hostname = computed(() => {
  return latestResult.value?.hostname || null
})

const hasPeriodData = computed(() => {
  return configuredPeriod.value && periodData.value && periodData.value.results && periodData.value.results.length > 0
})

const displayStartTime = computed(() => {
  if (configuredPeriod.value) {
    if (periodLoading.value || !periodData.value) return ''
    // Use the configured period duration as the start time
    const match = configuredPeriod.value.match(/^(\d+)([hd])$/)
    if (match) {
      const value = parseInt(match[1])
      const unit = match[2]
      const ms = unit === 'd' ? value * 24 * 60 * 60 * 1000 : value * 60 * 60 * 1000
      return generatePrettyTimeAgo(new Date(Date.now() - ms).toISOString())
    }
    return ''
  }
  // Non-period: use oldest displayed result
  if (!props.endpoint.results || props.endpoint.results.length === 0) return ''
  const oldestIndex = Math.max(0, props.endpoint.results.length - props.maxResults)
  return generatePrettyTimeAgo(props.endpoint.results[oldestIndex].timestamp)
})

const displayEndTime = computed(() => {
  if (configuredPeriod.value) {
    if (periodLoading.value || !periodData.value) return ''
    const results = periodData.value.results
    if (!results || results.length === 0) return ''
    return generatePrettyTimeAgo(results[results.length - 1].timestamp)
  }
  // Non-period: use newest result
  if (!props.endpoint.results || props.endpoint.results.length === 0) return ''
  return generatePrettyTimeAgo(props.endpoint.results[props.endpoint.results.length - 1].timestamp)
})

const displayUptime = computed(() => {
  if (configuredPeriod.value) {
    if (periodLoading.value || !periodData.value) return null
    return periodData.value.uptime
  }
  // Non-period: use uptime from the API response (day uptime as default)
  if (props.endpoint.uptime) {
    return props.endpoint.uptime.day
  }
  // Fallback: compute from displayed results
  const results = props.endpoint.results
  if (!results || results.length === 0) return null
  let success = 0
  let total = 0
  for (const r of results) {
    if (r) {
      total++
      if (r.success) success++
    }
  }
  if (total === 0) return null
  return success / total
})

const displayBars = computed(() => {
  if (hasPeriodData.value) {
    return periodData.value.results
  }
  const results = [...(props.endpoint.results || [])]
  while (results.length < props.maxResults) {
    results.unshift(null)
  }
  return results.slice(-props.maxResults)
})

const barClickable = (item) => {
  return item !== null && item !== undefined
}

const barClass = (item, index) => {
  if (!item) return 'bg-gray-200 dark:bg-gray-700'
  if (item.missing) return 'bg-gray-200 dark:bg-gray-700'
  const success = item.success
  const selected = selectedResultIndex.value === index
  if (success) {
    return selected ? 'bg-green-700' : 'bg-green-500 hover:bg-green-700'
  }
  return selected ? 'bg-red-700' : 'bg-red-500 hover:bg-red-700'
}

const formattedResponseTime = computed(() => {
  if (configuredPeriod.value) {
    if (periodLoading.value || !periodData.value) return ''
  }
  const source = hasPeriodData.value ? periodData.value.results : props.endpoint.results
  if (!source || source.length === 0) return 'N/A'
  
  let total = 0
  let count = 0
  let min = Infinity
  let max = 0
  
  for (const result of source) {
    if (!result || result.missing) continue
    const durationMs = (result.duration || 0) / 1000000
    if (durationMs > 0) {
      total += durationMs
      count++
      min = Math.min(min, durationMs)
      max = Math.max(max, durationMs)
    }
  }
  
  if (count === 0) return 'N/A'
  
  if (props.showAverageResponseTime) {
    const avgMs = Math.round(total / count)
    return `~${avgMs}ms`
  } else {
    const minMs = Math.trunc(min)
    const maxMs = Math.trunc(max)
    if (minMs === maxMs) {
      return `${minMs}ms`
    }
    return `${minMs}-${maxMs}ms`
  }
})

const formatUptimePercent = (value) => {
  if (value === undefined || value === null) return 'N/A'
  return (value * 100).toFixed(2).replace(/\.?0+$/, '') + '%'
}

const uptimeColor = (value) => {
  if (value === undefined || value === null) return ''
  if (value >= 0.975) return 'text-green-500'
  if (value >= 0.95) return 'text-green-400'
  if (value >= 0.9) return 'text-yellow-500'
  if (value >= 0.8) return 'text-orange-500'
  return 'text-red-500'
}

const navigateToDetails = () => {
  router.push(`/endpoints/${props.endpoint.key}`)
}

const handleMouseEnter = (result, event) => {
  emit('showTooltip', result, event, 'hover')
}

const handleMouseLeave = (result, event) => {
  emit('showTooltip', null, event, 'hover')
}

const handleClick = (result, event, index) => {
  window.dispatchEvent(new CustomEvent('clear-data-point-selection'))
  if (selectedResultIndex.value === index) {
    selectedResultIndex.value = null
    emit('showTooltip', null, event, 'click')
  } else {
    selectedResultIndex.value = index
    emit('showTooltip', result, event, 'click')
  }
}

const handleClearSelection = () => {
  selectedResultIndex.value = null
}

const fetchPeriodData = async () => {
  if (!configuredPeriod.value) {
    periodData.value = null
    return
  }
  periodLoading.value = true
  try {
    const parts = props.maxResults
    const response = await fetch(`/api/v1/endpoints/${props.endpoint.key}/period-statuses/${configuredPeriod.value}/${parts}`)
    if (response.status === 200) {
      periodData.value = await response.json()
    }
  } catch (e) {
    // Silently fail, will fall back to regular results
  } finally {
    periodLoading.value = false
  }
}

onMounted(() => {
  window.addEventListener('clear-data-point-selection', handleClearSelection)
  fetchPeriodData()
})

onUnmounted(() => {
  window.removeEventListener('clear-data-point-selection', handleClearSelection)
})

watch(() => [props.endpoint.key, configuredPeriod.value], () => {
  fetchPeriodData()
})
</script>
