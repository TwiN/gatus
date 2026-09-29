<template>
  <div :class="combineClasses('flex flex-col space-y-1.5 p-6', $attrs.class ?? '')">
    <slot />
  </div>
</template>

<script setup>
import { combineClasses } from '@/utils/misc'
</script>