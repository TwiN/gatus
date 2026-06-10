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
        v-for="(result, index) in data.results"
        :key="index"
        :class="[
          'flex-1 h-6 sm:h-8 rounded-sm transition-all cursor-pointer',
          result.missing ? 'bg-gray-200 dark:bg-gray-700' : (result.success ? 'bg-green-500 hover:bg-green-700' : 'bg-red-500 hover:bg-red-700')
        ]"
        @mouseenter="handleMouseEnter(result, $event)"
        @mouseleave="handleMouseLeave($event)"
        @click.stop="handleClick(result, $event)"
      />
    </div>
    <div class="flex items-center gap-1 text-xs text-muted-foreground mt-1">
      <span>{{ periodStartTime }}</span>
      <span class="flex-1 border-t border-dashed border-muted-foreground/30 mx-1"></span>
      <span class="font-medium" :class="uptimeColor(data.uptime)">{{ formatUptimePercent(data.uptime) }} uptime</span>
      <span class="flex-1 border-t border-dashed border-muted-foreground/30 mx-1"></span>
      <span>{{ periodEndTime }}</span>
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

const periodStartTime = computed(() => {
  if (!data.value || !data.value.results || data.value.results.length === 0) return ''
  const first = data.value.results[0]
  if (!first || first.missing) return ''
  return generatePrettyTimeAgo(first.timestamp)
})

const periodEndTime = computed(() => {
  if (!data.value || !data.value.results || data.value.results.length === 0) return ''
  const results = data.value.results
  const last = results[results.length - 1]
  if (!last) return ''
  return generatePrettyTimeAgo(last.timestamp)
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

const buildTooltipData = (result) => {
  if (result.missing) {
    return {
      timestamp: result.timestamp,
      duration: 0,
      success: false,
      conditionResults: [{ condition: 'No data available', success: false }],
      errors: [],
      hostname: '',
      name: '',
    }
  }
  return {
    timestamp: result.timestamp,
    duration: result.duration,
    success: result.success,
    conditionResults: result.conditionResults || [],
    errors: result.errors || [],
    hostname: result.hostname || '',
    name: '',
  }
}

const handleMouseEnter = (result, event) => {
  emit('showTooltip', buildTooltipData(result), event, 'hover')
}

const handleMouseLeave = (event) => {
  emit('showTooltip', null, event, 'hover')
}

const handleClick = (result, event) => {
  emit('showTooltip', buildTooltipData(result), event, 'click')
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
