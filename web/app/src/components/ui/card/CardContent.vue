<template>
  <div :class="combineClasses('p-6 pt-0', $attrs.class ?? '')">
    <slot />
  </div>
</template>

<script setup>
import { combineClasses } from '@/utils/misc'
</script>