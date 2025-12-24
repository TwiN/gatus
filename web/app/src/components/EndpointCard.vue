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
          <div class="flex gap-0.5">
            <div
              v-for="(result, index) in displayResults"
              :key="index"
              :class="[
                'flex-1 h-6 sm:h-8 rounded-sm transition-all',
                result ? 'cursor-pointer' : '',
                result ? (
                  result.success 
                    ? (selectedResultIndex === index ? 'bg-green-700' : 'bg-green-500 hover:bg-green-700')
                    : (selectedResultIndex === index ? 'bg-red-700' : 'bg-red-500 hover:bg-red-700')
                ) : 'bg-gray-200 dark:bg-gray-700'
              ]"
              @mouseenter="result && handleMouseEnter(result, $event)"
              @mouseleave="result && handleMouseLeave(result, $event)"
              @click.stop="result && handleClick(result, $event, index)"
            />
          </div>
          <div class="flex items-center justify-between text-xs text-muted-foreground mt-1">
            <span>{{ oldestResultTime }}</span>
            <span>{{ newestResultTime }}</span>
          </div>
        </div>
      </div>
    </CardContent>
  </Card>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
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

// Track selected data point
const selectedResultIndex = ref(null)

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

const displayResults = computed(() => {
  const results = [...(props.endpoint.results || [])]
  while (results.length < props.maxResults) {
    results.unshift(null)
  }
  return results.slice(-props.maxResults)
})

const formattedResponseTime = computed(() => {
  if (!props.endpoint.results || props.endpoint.results.length === 0) {
    return 'N/A'
  }
  
  let total = 0
  let count = 0
  let min = Infinity
  let max = 0
  
  for (const result of props.endpoint.results) {
    if (result.duration) {
      const durationMs = result.duration / 1000000
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
    // Show min-max range
    const minMs = Math.trunc(min)
    const maxMs = Math.trunc(max)
    // If min and max are the same, show single value
    if (minMs === maxMs) {
      return `${minMs}ms`
    }
    return `${minMs}-${maxMs}ms`
  }
})

const oldestResultTime = computed(() => {
  if (!props.endpoint.results || props.endpoint.results.length === 0) return ''
  const oldestResultIndex = Math.max(0, props.endpoint.results.length - props.maxResults)
  return generatePrettyTimeAgo(props.endpoint.results[oldestResultIndex].timestamp)
})

const newestResultTime = computed(() => {
  if (!props.endpoint.results || props.endpoint.results.length === 0) return ''
  return generatePrettyTimeAgo(props.endpoint.results[props.endpoint.results.length - 1].timestamp)
})

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
  // Clear selections in other cards first
  window.dispatchEvent(new CustomEvent('clear-data-point-selection'))
  // Then toggle this card's selection
  if (selectedResultIndex.value === index) {
    selectedResultIndex.value = null
    emit('showTooltip', null, event, 'click')
  } else {
    selectedResultIndex.value = index
    emit('showTooltip', result, event, 'click')
  }
}

// Listen for clear selection event
const handleClearSelection = () => {
  selectedResultIndex.value = null
}

onMounted(() => {
  window.addEventListener('clear-data-point-selection', handleClearSelection)
})

onUnmounted(() => {
  window.removeEventListener('clear-data-point-selection', handleClearSelection)
})
</script>