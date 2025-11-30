<template>
  <div class="flex justify-center items-center">
    <img 
      :class="[
        'animate-spin rounded-full opacity-60 grayscale',
        sizeClass,
      ]"
      src="../assets/logo.svg" 
      alt="Gatus logo" 
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  size: {
    type: String,
    default: 'md',
    validator: (value) => ['xs', 'sm', 'md', 'lg', 'xl'].includes(value)
  },
})

const sizeClass = computed(() => {
  const sizes = {
    xs: 'w-4 h-4',
    sm: 'w-6 h-6',
    md: 'w-8 h-8',
    lg: 'w-12 h-12',
    xl: 'w-16 h-16'
  }
  return sizes[props.size] || sizes.md
})
</script>