<template>
    <Badge :style="`background-color: ${color};`" class="flex items-center gap-1 text-white">
      <span :style="`background-color: ${color}; filter: brightness(115%)`" class="w-2 h-2 rounded-full"></span>
      {{ label }}
    </Badge>
</template>

<script setup>
import { computed } from 'vue'
import { Badge } from '@/components/ui/badge'
import { getStateColor } from '@/utils/color'

const props = defineProps({
  status: {
    type: String,
    required: true,
  },
})

const label = computed(() => {
  return props.status.charAt(0).toUpperCase() + props.status.slice(1).replace(/_/g, ' ')
})

const color = computed(() => {
  return getStateColor(props.status)
})
</script>