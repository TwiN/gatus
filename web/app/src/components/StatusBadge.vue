<template>
  <Badge :variant="variant" class="flex items-center gap-1">
    <span :class="['w-2 h-2 rounded-full', dotClass]"></span>
    {{ label }}
  </Badge>
</template>

<script setup>
import { computed } from 'vue'
import { Badge } from '@/components/ui/badge'

const props = defineProps({
  status: {
    type: String,
    required: true,
    validator: (value) => ['healthy', 'unhealthy', 'degraded', 'unknown'].includes(value)
  }
})

const variant = computed(() => {
  switch (props.status) {
    case 'healthy':
      return 'success'
    case 'unhealthy':
      return 'destructive'
    case 'degraded':
      return 'warning'
    default:
      return 'secondary'
  }
})

const label = computed(() => {
  switch (props.status) {
    case 'healthy':
      return 'Healthy'
    case 'unhealthy':
      return 'Unhealthy'
    case 'degraded':
      return 'Degraded'
    default:
      return 'Unknown'
  }
})

const dotClass = computed(() => {
  switch (props.status) {
    case 'healthy':
      return 'bg-green-400'
    case 'unhealthy':
      return 'bg-red-400'
    case 'degraded':
      return 'bg-yellow-400'
    default:
      return 'bg-gray-400'
  }
})
</script>