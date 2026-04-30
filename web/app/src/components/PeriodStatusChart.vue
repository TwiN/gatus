<template>
  <div v-if="loading" class="flex items-center justify-center py-8">
    <Loading size="sm" />
  </div>
  <div v-else-if="error" class="text-center py-4 text-muted-foreground text-sm">
    {{ error }}
  </div>
  <div v-else-if="data" class="space-y-2">
    <div class="flex gap-0.5">
      <div
        v-for="(slice, index) in data.slices"
        :key="index"
        :class="[
          'flex-1 h-6 sm:h-8 rounded-sm transition-all cursor-pointer',
          getSliceColor(slice.uptime)
        ]"
        @mouseenter="handleMouseEnter(slice, $event)"
        @mouseleave="handleMouseLeave($event)"
        @click.stop="handleClick(slice, $event)"
      />
    </div>
    <div class="flex items-center justify-between text-xs text-muted-foreground">
      <span>{{ formatTimestamp(data.slices[0]?.timestamp) }}</span>
      <span>{{ formatTimestamp(data.slices[data.slices.length - 1]?.timestamp) }}</span>
    </div>
    <div class="text-center text-sm text-muted-foreground pt-1">
      <span>Uptime over {{ data.duration }}: </span>
      <span class="font-medium" :class="overallUptimeColor">{{ overallUptimePercent }}%</span>
      <span class="mx-2">|</span>
      <span>Avg response: </span>
      <span class="font-medium">{{ overallAvgResponseTime }}ms</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import Loading from '@/components/Loading.vue'
import { generatePrettyTimeAgo } from '@/utils/time'

const props = defineProps({
  endpointKey: {
    type: String,
    required: true
  },
  duration: {
    type: String,
    required: true
  },
  parts: {
    type: Number,
    default: 50
  }
})

const emit = defineEmits(['showTooltip'])

const data = ref(null)
const loading = ref(false)
const error = ref(null)

const overallUptimePercent = computed(() => {
  if (!data.value || !data.value.slices || data.value.slices.length === 0) return 'N/A'
  let totalUptime = 0
  let count = 0
  for (const slice of data.value.slices) {
    if (slice.uptime > 0 || slice.response_time > 0) {
      totalUptime += slice.uptime
      count++
    }
  }
  if (count === 0) return 'N/A'
  const avg = totalUptime / count
  return (avg * 100).toFixed(2).replace(/\.?0+$/, '')
})

const overallUptimeColor = computed(() => {
  if (!data.value || !data.value.slices) return ''
  const val = parseFloat(overallUptimePercent.value)
  if (isNaN(val)) return ''
  if (val >= 97.5) return 'text-green-500'
  if (val >= 95) return 'text-green-400'
  if (val >= 90) return 'text-yellow-400'
  if (val >= 80) return 'text-orange-400'
  return 'text-red-500'
})

const overallAvgResponseTime = computed(() => {
  if (!data.value || !data.value.slices || data.value.slices.length === 0) return 'N/A'
  let total = 0
  let count = 0
  for (const slice of data.value.slices) {
    if (slice.response_time > 0) {
      total += slice.response_time
      count++
    }
  }
  if (count === 0) return 'N/A'
  return Math.round(total / count)
})

const getSliceColor = (uptime) => {
  if (uptime >= 0.975) return 'bg-green-500 hover:bg-green-700'
  if (uptime >= 0.95) return 'bg-green-400 hover:bg-green-600'
  if (uptime >= 0.9) return 'bg-yellow-400 hover:bg-yellow-600'
  if (uptime >= 0.8) return 'bg-orange-400 hover:bg-orange-600'
  if (uptime >= 0.65) return 'bg-orange-500 hover:bg-orange-700'
  return 'bg-red-500 hover:bg-red-700'
}

const formatTimestamp = (ts) => {
  if (!ts) return ''
  return generatePrettyTimeAgo(new Date(ts).toISOString())
}

const handleMouseEnter = (slice, event) => {
  const tooltipData = {
    timestamp: new Date(slice.timestamp).toISOString(),
    duration: 0,
    success: slice.uptime >= 0.99,
    conditionResults: [
      { condition: `Uptime: ${(slice.uptime * 100).toFixed(2)}%`, success: slice.uptime >= 0.99 },
      { condition: `Avg Response: ${slice.response_time}ms`, success: true },
    ],
    errors: [],
    hostname: '',
    name: '',
  }
  emit('showTooltip', tooltipData, event, 'hover')
}

const handleMouseLeave = (event) => {
  emit('showTooltip', null, event, 'hover')
}

const handleClick = (slice, event) => {
  const tooltipData = {
    timestamp: new Date(slice.timestamp).toISOString(),
    duration: 0,
    success: slice.uptime >= 0.99,
    conditionResults: [
      { condition: `Uptime: ${(slice.uptime * 100).toFixed(2)}%`, success: slice.uptime >= 0.99 },
      { condition: `Avg Response: ${slice.response_time}ms`, success: true },
    ],
    errors: [],
    hostname: '',
    name: '',
  }
  emit('showTooltip', tooltipData, event, 'click')
}

const fetchData = async () => {
  loading.value = true
  error.value = null
  try {
    const response = await fetch(`/api/v1/endpoints/${props.endpointKey}/period-statuses/${props.duration}/${props.parts}`)
    if (response.status === 200) {
      data.value = await response.json()
    } else {
      error.value = await response.text()
    }
  } catch (e) {
    error.value = 'Failed to load period data'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchData()
})

watch(() => [props.endpointKey, props.duration, props.parts], () => {
  fetchData()
})
</script>
