<template>
  <Card class="suite h-full flex flex-col transition hover:shadow-lg hover:scale-[1.01] dark:hover:border-gray-700">
    <CardHeader class="suite-header px-3 sm:px-6 pt-3 sm:pt-6 pb-2 space-y-0">
      <div class="flex items-start justify-between gap-2 sm:gap-3">
        <div class="flex-1 min-w-0 overflow-hidden">
          <CardTitle class="text-base sm:text-lg truncate">
            <span 
              class="hover:text-primary cursor-pointer hover:underline text-sm sm:text-base block truncate" 
              @click="navigateToDetails" 
              @keydown.enter="navigateToDetails"
              :title="suite.name"
              role="link"
              tabindex="0"
              :aria-label="`View details for suite ${suite.name}`">
              {{ suite.name }}
            </span>
          </CardTitle>
          <div class="flex items-center gap-2 text-xs sm:text-sm text-muted-foreground">
            <span v-if="suite.group" class="truncate" :title="suite.group">{{ suite.group }}</span>
            <span v-if="suite.group && endpointCount">•</span>
            <span v-if="endpointCount">{{ endpointCount }} endpoint{{ endpointCount !== 1 ? 's' : '' }}</span>
          </div>
        </div>
        <div class="flex-shrink-0 ml-2">
          <StatusBadge :status="currentStatus" />
        </div>
      </div>
    </CardHeader>
    <CardContent class="suite-content flex-1 pb-3 sm:pb-4 px-3 sm:px-6 pt-2">
      <div class="space-y-2">
        <div>
          <div class="flex items-center justify-between mb-1">
            <p class="text-xs text-muted-foreground">Success Rate: {{ successRate }}%</p>
            <p class="text-xs text-muted-foreground" v-if="averageDuration !== null">{{ averageDuration }}ms avg</p>
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
  suite: {
    type: Object,
    required: true
  },
  maxResults: {
    type: Number,
    default: 50
  }
})

const emit = defineEmits(['showTooltip'])

// Track selected data point
const selectedResultIndex = ref(null)

// Computed properties
const displayResults = computed(() => {
  const results = [...(props.suite.results || [])]
  while (results.length < props.maxResults) {
    results.unshift(null)
  }
  return results.slice(-props.maxResults)
})

const currentStatus = computed(() => {
  if (!props.suite.results || props.suite.results.length === 0) {
    return 'unknown'
  }
  return props.suite.results[props.suite.results.length - 1].success ? 'healthy' : 'unhealthy'
})

const endpointCount = computed(() => {
  if (!props.suite.results || props.suite.results.length === 0) {
    return 0
  }
  const latestResult = props.suite.results[props.suite.results.length - 1]
  return latestResult.endpointResults ? latestResult.endpointResults.length : 0
})

const successRate = computed(() => {
  if (!props.suite.results || props.suite.results.length === 0) {
    return 0
  }
  
  const successful = props.suite.results.filter(r => r.success).length
  return Math.round((successful / props.suite.results.length) * 100)
})

const averageDuration = computed(() => {
  if (!props.suite.results || props.suite.results.length === 0) {
    return null
  }
  
  const total = props.suite.results.reduce((sum, r) => sum + (r.duration || 0), 0)
  // Convert nanoseconds to milliseconds
  return Math.trunc((total / props.suite.results.length) / 1000000)
})

const oldestResultTime = computed(() => {
  if (!props.suite.results || props.suite.results.length === 0) {
    return 'N/A'
  }
  
  const oldestResult = props.suite.results[0]
  return generatePrettyTimeAgo(oldestResult.timestamp)
})

const newestResultTime = computed(() => {
  if (!props.suite.results || props.suite.results.length === 0) {
    return 'Now'
  }
  
  const newestResult = props.suite.results[props.suite.results.length - 1]
  return generatePrettyTimeAgo(newestResult.timestamp)
})

// Methods
const navigateToDetails = () => {
  router.push(`/suites/${props.suite.key}`)
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

<style scoped>
.suite {
  transition: all 0.2s ease;
}

.suite:hover {
  transform: translateY(-2px);
}

.suite-header {
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
}

.dark .suite-header {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}
</style>