<template>
  <div class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors
              focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 border-transparent bg-primary
              text-primary-foreground hover:bg-primary/80 flex items-center gap-1 text-white"
       :style="`background-color: ${color};`">
    <span :style="`background-color: ${color}; filter: brightness(115%)`" class="w-2 h-2 rounded-full"></span>
    {{ label }}
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { getStateColor } from '@/utils/color'

const props = defineProps({
  status: {
    type: String,
    required: true,
  },
})

const label = computed(() => {
  if (!props.status) return 'Unknown'
  return props.status.charAt(0).toUpperCase() + props.status.slice(1).replace(/_/g, ' ') // TODO#227 Capitalize every word
})

const color = computed(() => {
  if (!props.status) return window.config?.localStateColors.nodata
  return getStateColor(props.status)
})
</script>