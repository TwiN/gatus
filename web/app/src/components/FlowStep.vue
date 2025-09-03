<template>
  <div class="flex items-start gap-4 relative group hover:bg-accent/30 rounded-lg p-2 -m-2 transition-colors cursor-pointer"
       @click="$emit('step-click')">
    <!-- Step circle with status icon -->
    <div class="relative flex-shrink-0">
      <!-- Connection line from previous step -->
      <div v-if="index > 0" :class="incomingLineClasses" class="absolute left-1/2 bottom-8 w-0.5 h-4 -translate-x-px"></div>
      
      <div :class="circleClasses" class="w-8 h-8 rounded-full flex items-center justify-center">
        <component :is="statusIcon" class="w-4 h-4" />
      </div>
      
      <!-- Connection line to next step -->
      <div v-if="!isLast" :class="connectionLineClasses" class="absolute left-1/2 top-8 w-0.5 h-4 -translate-x-px"></div>
    </div>
    
    <!-- Step content -->
    <div class="flex-1 min-w-0 pt-1">
      <div class="flex items-center justify-between gap-2 mb-1">
        <h4 class="font-medium text-sm truncate">{{ step.name }}</h4>
        <span class="text-xs text-muted-foreground whitespace-nowrap">
          {{ formatDuration(step.duration) }}
        </span>
      </div>
      
      <!-- Step badges -->
      <div class="flex flex-wrap gap-1">
        <span v-if="step.isAlwaysRun" class="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200 rounded-md">
          <RotateCcw class="w-3 h-3" />
          Always Run
        </span>
        <span v-if="step.errors?.length" class="inline-flex items-center px-2 py-1 text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 rounded-md">
          {{ step.errors.length }} error{{ step.errors.length !== 1 ? 's' : '' }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { CheckCircle, XCircle, SkipForward, RotateCcw, Pause } from 'lucide-vue-next'
import { formatDuration } from '@/utils/format'

const props = defineProps({
  step: { type: Object, required: true },
  index: { type: Number, required: true },
  isLast: { type: Boolean, default: false },
  previousStep: { type: Object, default: null }
})

defineEmits(['step-click'])

// Status icon mapping
const statusIcon = computed(() => {
  switch (props.step.status) {
    case 'success': return CheckCircle
    case 'failed': return XCircle
    case 'skipped': return SkipForward
    case 'not-started': return Pause
    default: return Pause
  }
})

// Circle styling classes
const circleClasses = computed(() => {
  const baseClasses = 'border-2'
  
  if (props.step.isAlwaysRun) {
    // Always-run endpoints get a special ring effect
    switch (props.step.status) {
      case 'success':
        return `${baseClasses} bg-green-500 text-white border-green-600 ring-2 ring-blue-200 dark:ring-blue-800`
      case 'failed':
        return `${baseClasses} bg-red-500 text-white border-red-600 ring-2 ring-blue-200 dark:ring-blue-800`
      default:
        return `${baseClasses} bg-blue-500 text-white border-blue-600 ring-2 ring-blue-200 dark:ring-blue-800`
    }
  }
  
  switch (props.step.status) {
    case 'success':
      return `${baseClasses} bg-green-500 text-white border-green-600`
    case 'failed':
      return `${baseClasses} bg-red-500 text-white border-red-600`
    case 'skipped':
      return `${baseClasses} bg-gray-400 text-white border-gray-500`
    case 'not-started':
      return `${baseClasses} bg-gray-200 text-gray-500 border-gray-300 dark:bg-gray-700 dark:text-gray-400 dark:border-gray-600`
    default:
      return `${baseClasses} bg-gray-200 text-gray-500 border-gray-300 dark:bg-gray-700 dark:text-gray-400 dark:border-gray-600`
  }
})

// Incoming connection line styling (from previous step to this step)
const incomingLineClasses = computed(() => {
  if (!props.previousStep) return 'bg-gray-300 dark:bg-gray-600'
  
  // If this step is skipped, the line should be dashed/gray
  if (props.step.status === 'skipped') {
    return 'border-l-2 border-dashed border-gray-400 bg-transparent'
  }
  
  // Otherwise, color based on previous step's status
  switch (props.previousStep.status) {
    case 'success':
      return 'bg-green-500'
    case 'failed':
      // If previous failed but this ran (always-run), show red line
      return 'bg-red-500'
    default:
      return 'bg-gray-300 dark:bg-gray-600'
  }
})

// Outgoing connection line styling (from this step to next)
const connectionLineClasses = computed(() => {
  const nextStep = props.step.nextStepStatus
  switch (props.step.status) {
    case 'success':
      return nextStep === 'skipped' 
        ? 'bg-gray-300 dark:bg-gray-600' 
        : 'bg-green-500'
    case 'failed':
      return nextStep === 'skipped'
        ? 'border-l-2 border-dashed border-gray-400 bg-transparent'
        : 'bg-red-500'
    default:
      return 'bg-gray-300 dark:bg-gray-600'
  }
})

</script>