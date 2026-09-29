<template>
  <h3 :class="combineClasses('text-2xl font-semibold leading-none tracking-tight', $attrs.class ?? '')">
    <slot />
  </h3>
</template>

<script setup>
import { combineClasses } from '@/utils/misc'
</script>