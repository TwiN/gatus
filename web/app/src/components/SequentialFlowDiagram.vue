<template>
  <div class="space-y-4">
    <!-- Timeline header -->
    <div class="flex items-center gap-4">
      <div class="text-sm font-medium text-muted-foreground">Start</div>
      <div class="flex-1 h-1 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
        <div 
          class="h-full bg-green-500 dark:bg-green-600 rounded-full transition-all duration-300 ease-out"
          :style="{ width: progressPercentage + '%' }"
        ></div>
      </div>
      <div class="text-sm font-medium text-muted-foreground">End</div>
    </div>
    
    <!-- Progress stats -->
    <div class="flex items-center justify-between text-xs text-muted-foreground">
      <span>{{ completedSteps }}/{{ totalSteps }} steps successful</span>
      <span v-if="totalDuration > 0">{{ formatDuration(totalDuration) }} total</span>
    </div>
    
    <!-- Flow steps -->
    <div class="space-y-2">
      <FlowStep
        v-for="(step, index) in flowSteps"
        :key="index"
        :step="step"
        :index="index"
        :is-last="index === flowSteps.length - 1"
        :previous-step="index > 0 ? flowSteps[index - 1] : null"
        @step-click="$emit('step-selected', step, index)"
      />
    </div>
    
    <!-- Legend -->
    <div class="mt-6 pt-4 border-t">
      <div class="text-sm font-medium text-muted-foreground mb-2">Status Legend</div>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
        <div v-if="hasSuccessSteps" class="flex items-center gap-2">
          <div class="w-4 h-4 rounded-full bg-green-500 flex items-center justify-center">
            <CheckCircle class="w-3 h-3 text-white" />
          </div>
          <span class="text-muted-foreground">Success</span>
        </div>
        
        <div v-if="hasFailedSteps" class="flex items-center gap-2">
          <div class="w-4 h-4 rounded-full bg-red-500 flex items-center justify-center">
            <XCircle class="w-3 h-3 text-white" />
          </div>
          <span class="text-muted-foreground">Failed</span>
        </div>
        
        <div v-if="hasSkippedSteps" class="flex items-center gap-2">
          <div class="w-4 h-4 rounded-full bg-gray-400 flex items-center justify-center">
            <SkipForward class="w-3 h-3 text-white" />
          </div>
          <span class="text-muted-foreground">Skipped</span>
        </div>
        
        <div v-if="hasAlwaysRunSteps" class="flex items-center gap-2">
          <div class="w-4 h-4 rounded-full bg-blue-500 border-2 border-blue-200 dark:border-blue-800 flex items-center justify-center">
            <RotateCcw class="w-3 h-3 text-white" />
          </div>
          <span class="text-muted-foreground">Always Run</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { CheckCircle, XCircle, SkipForward, RotateCcw } from 'lucide-vue-next'
import FlowStep from './FlowStep.vue'
import { formatDuration } from '@/utils/format'

const props = defineProps({
  flowSteps: {
    type: Array,
    default: () => []
  },
  progressPercentage: {
    type: Number,
    default: 0
  },
  completedSteps: {
    type: Number,
    default: 0
  },
  totalSteps: {
    type: Number,
    default: 0
  }
})

defineEmits(['step-selected'])

// Use props instead of computing locally for consistency
const completedSteps = computed(() => props.completedSteps)
const totalSteps = computed(() => props.totalSteps)

const totalDuration = computed(() => {
  return props.flowSteps.reduce((total, step) => {
    return total + (step.duration || 0)
  }, 0)
})

// Legend visibility computed properties
const hasSuccessSteps = computed(() => {
  return props.flowSteps.some(step => step.status === 'success')
})

const hasFailedSteps = computed(() => {
  return props.flowSteps.some(step => step.status === 'failed')
})

const hasSkippedSteps = computed(() => {
  return props.flowSteps.some(step => step.status === 'skipped')
})

const hasAlwaysRunSteps = computed(() => {
  return props.flowSteps.some(step => step.isAlwaysRun === true)
})

</script>