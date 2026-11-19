<template>
  <div :class="combineClasses(badgeVariants({ variant }), $attrs.class ?? '')">
    <slot />
  </div>
</template>

<script setup>
import { cva } from 'class-variance-authority'
import { combineClasses } from '@/utils/misc'

defineProps({
  variant: {
    type: String,
    default: 'default',
  },
})

const badgeVariants = cva(
  'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
  {
    variants: {
      variant: {
        default: 'border-transparent bg-primary text-primary-foreground hover:bg-primary/80',
        secondary: 'border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80',
        destructive: 'border-red-600/20 bg-red-50 text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-400',
        outline: 'text-foreground',
        success: 'border-green-600/20 bg-green-50 text-green-700 dark:border-green-500/20 dark:bg-green-500/10 dark:text-green-400',
        warning: 'border-yellow-600/20 bg-yellow-50 text-yellow-800 dark:border-yellow-500/20 dark:bg-yellow-500/10 dark:text-yellow-400',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)
</script>