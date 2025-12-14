<template>
    <Badge :style="`background-color: ${primaryColor};`" class="flex items-center gap-1 text-white">
      <span :style="`background-color: ${primaryColor}; filter: brightness(115%)`" class="w-2 h-2 rounded-full"></span>
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
  },
  color: {
    type: String,
    default: null,
  },
})

const label = computed(() => {
  return props.status.charAt(0).toUpperCase() + props.status.slice(1).replace(/_/g, ' ')
})

const primaryColor = computed(() => {
  return props.color ?? '#374151'
})
</script>