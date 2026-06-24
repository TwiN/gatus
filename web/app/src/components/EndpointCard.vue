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
          <div class="flex items-center justify-between mb-1.5">
            <span
              v-if="uptimePercentage !== null"
              class="text-xs font-semibold tabular-nums"
              :class="uptimePercentage >= 99 ? 'text-green-500' : uptimePercentage >= 95 ? 'text-amber-500' : 'text-red-500'"
            >{{ uptimePercentage }}%</span>
            <div v-else class="flex-1"></div>
            <p class="text-xs text-muted-foreground" :title="showAverageResponseTime ? 'Average response time' : 'Minimum and maximum response time'">{{ formattedResponseTime }}</p>
          </div>
          <div class="overflow-hidden rounded-lg h-7 sm:h-9" style="display:grid;grid-template-columns:repeat(40,1fr);gap:1px;">
            <div
              v-for="(result, index) in displayBuckets"
              :key="index"
              :class="[
                'transition-opacity',
                result ? 'cursor-pointer' : '',
                result ? (
                  result.mixed
                    ? (selectedResultIndex === index ? 'bg-amber-500' : 'bg-amber-400 hover:bg-amber-500')
                    : result.success
                      ? (selectedResultIndex === index ? 'bg-green-600' : 'bg-green-500 hover:bg-green-600')
                      : (selectedResultIndex === index ? 'bg-red-600' : 'bg-red-500 hover:bg-red-600')
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

const DISPLAY_BUCKETS = 40

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

const latestResult = computed(() => {
  if (!props.endpoint.results || props.endpoint.results.length === 0) return null
  return props.endpoint.results[props.endpoint.results.length - 1]
})

const currentStatus = computed(() => {
  if (!latestResult.value) return 'unknown'
  return latestResult.value.success ? 'healthy' : 'unhealthy'
})

const hostname = computed(() => {
  return latestResult.value?.hostname || null
})

// Aggregate raw results into DISPLAY_BUCKETS for a clean visual
const displayBuckets = computed(() => {
  const results = props.endpoint.results || []
  const total = props.maxResults

  // Pad with nulls at the start so index 0 = oldest slot
  const padded = Array(Math.max(0, total - results.length)).fill(null).concat(results.slice(-total))

  const buckets = []
  for (let i = 0; i < DISPLAY_BUCKETS; i++) {
    const start = Math.floor(i * total / DISPLAY_BUCKETS)
    const end = Math.floor((i + 1) * total / DISPLAY_BUCKETS)
    const slice = padded.slice(start, end)
    const nonNull = slice.filter(Boolean)

    if (nonNull.length === 0) {
      buckets.push(null)
    } else {
      const successCount = nonNull.filter(r => r.success).length
      const latest = nonNull[nonNull.length - 1]
      buckets.push({
        ...latest,
        success: successCount === nonNull.length,
        mixed: successCount > 0 && successCount < nonNull.length,
      })
    }
  }
  return buckets
})

const uptimePercentage = computed(() => {
  const results = props.endpoint.results || []
  if (results.length === 0) return null
  const successCount = results.filter(r => r.success).length
  const pct = (successCount / results.length) * 100
  return pct % 1 === 0 ? pct : Math.round(pct * 10) / 10
})

const formattedResponseTime = computed(() => {
  if (!props.endpoint.results || props.endpoint.results.length === 0) return 'N/A'

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
    return `~${Math.round(total / count)}ms`
  } else {
    const minMs = Math.trunc(min)
    const maxMs = Math.trunc(max)
    return minMs === maxMs ? `${minMs}ms` : `${minMs}-${maxMs}ms`
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

onMounted(() => {
  window.addEventListener('clear-data-point-selection', handleClearSelection)
})

onUnmounted(() => {
  window.removeEventListener('clear-data-point-selection', handleClearSelection)
})
</script>
