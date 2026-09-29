<template>
  <input
    :class="combineClasses(
      'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
      $attrs.class ?? ''
    )"
    :value="modelValue"
    @input="$emit('update:modelValue', $event.target.value)"
  />
</template>

<script setup>
import { combineClasses } from '@/utils/misc'

defineProps({
  modelValue: {
    type: [String, Number],
    default: '',
  },
})

defineEmits(['update:modelValue'])
</script>