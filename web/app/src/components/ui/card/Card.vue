<template>
  <div :class="combineClasses('rounded-lg border bg-card text-card-foreground shadow-sm', $attrs.class ?? '')">
    <slot />
  </div>
</template>

<script setup>
import { combineClasses } from '@/utils/misc'
</script>