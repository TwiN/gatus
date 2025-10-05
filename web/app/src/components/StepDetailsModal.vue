<template>
  <!-- Modal backdrop -->
  <div class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4 z-50" @click="$emit('close')">
    <!-- Modal content -->
    <div class="bg-background border rounded-lg shadow-lg max-w-2xl w-full max-h-[80vh] overflow-hidden" @click.stop>
      <!-- Header -->
      <div class="flex items-center justify-between p-4 border-b">
        <div>
          <h2 class="text-lg font-semibold flex items-center gap-2">
            <component :is="statusIcon" :class="iconClasses" class="w-5 h-5" />
            {{ step.name }}
          </h2>
          <p class="text-sm text-muted-foreground mt-1">
            Step {{ index + 1 }} • {{ formatDuration(step.duration) }}
          </p>
        </div>
        <Button variant="ghost" size="icon" @click="$emit('close')">
          <X class="w-4 h-4" />
        </Button>
      </div>
      
      <!-- Content -->
      <div class="p-4 space-y-4 overflow-y-auto max-h-[60vh]">
        <!-- Special properties -->
        <div v-if="step.isAlwaysRun" class="flex flex-wrap gap-2">
          <div class="flex items-center gap-2 px-3 py-2 bg-blue-50 dark:bg-blue-900/30 rounded-lg border border-blue-200 dark:border-blue-700">
            <RotateCcw class="w-4 h-4 text-blue-600 dark:text-blue-400" />
            <div>
              <p class="text-sm font-medium text-blue-900 dark:text-blue-200">Always Run</p>
              <p class="text-xs text-blue-600 dark:text-blue-400">This endpoint is configured to execute even after failures</p>
            </div>
          </div>
        </div>
        
        <!-- Errors section -->
        <div v-if="step.errors?.length" class="space-y-2">
          <h3 class="text-sm font-medium flex items-center gap-2 text-red-600 dark:text-red-400">
            <AlertCircle class="w-4 h-4" />
            Errors ({{ step.errors.length }})
          </h3>
          <div class="space-y-2">
            <div v-for="(error, index) in step.errors" :key="index" 
                 class="p-3 bg-red-50 dark:bg-red-900/50 border border-red-200 dark:border-red-700 rounded text-sm font-mono text-red-800 dark:text-red-300 break-all">
              {{ error }}
            </div>
          </div>
        </div>
        
        <!-- Timestamp -->
        <div v-if="step.result && step.result.timestamp" class="space-y-2">
          <h3 class="text-sm font-medium flex items-center gap-2">
            <Clock class="w-4 h-4" />
            Timestamp
          </h3>
          <p class="text-xs font-mono text-muted-foreground">{{ prettifyTimestamp(step.result.timestamp) }}</p>
        </div>
        
        <!-- Response details -->
        <div v-if="step.result" class="space-y-2">
          <h3 class="text-sm font-medium flex items-center gap-2">
            <Download class="w-4 h-4" />
            Response
          </h3>
          <div class="grid grid-cols-2 gap-4 text-xs">
            <div>
              <span class="text-muted-foreground">Duration:</span>
              <p class="font-mono mt-1">{{ formatDuration(step.result.duration) }}</p>
            </div>
            <div>
              <span class="text-muted-foreground">Success:</span>
              <p class="mt-1" :class="step.result.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">
                {{ step.result.success ? 'Yes' : 'No' }}
              </p>
            </div>
          </div>
        </div>

        <!-- Condition Results -->
        <div v-if="step.result?.conditionResults?.length" class="space-y-2">
          <h3 class="text-sm font-medium flex items-center gap-2">
            <CheckCircle class="w-4 h-4" />
            Condition Results ({{ step.result.conditionResults.length }})
          </h3>
          <div class="space-y-2 max-h-48 overflow-y-auto">
            <div
              v-for="(conditionResult, index) in step.result.conditionResults"
              :key="index"
              class="flex items-start gap-3 p-1 rounded-lg border"
              :class="conditionResult.success
                ? 'bg-green-50 dark:bg-green-900/30 border-green-200 dark:border-green-700'
                : 'bg-red-50 dark:bg-red-900/30 border-red-200 dark:border-red-700'"
            >
              <!-- Status icon -->
              <div class="flex-shrink-0 mt-0.5">
                <CheckCircle
                  v-if="conditionResult.success"
                  class="w-4 h-4 text-green-600 dark:text-green-400"
                />
                <XCircle
                  v-else
                  class="w-4 h-4 text-red-600 dark:text-red-400"
                />
              </div>

              <!-- Condition text -->
              <div class="flex-1 min-w-0 flex items-center justify-between gap-3">
                <p class="text-sm font-mono break-all"
                   :class="conditionResult.success
                     ? 'text-green-800 dark:text-green-200'
                     : 'text-red-800 dark:text-red-200'">
                  {{ conditionResult.condition }}
                </p>
                <span class="text-xs font-medium whitespace-nowrap"
                      :class="conditionResult.success
                        ? 'text-green-600 dark:text-green-400'
                        : 'text-red-600 dark:text-red-400'">
                  {{ conditionResult.success ? 'Passed' : 'Failed' }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- Endpoint Configuration -->
        <div v-if="step.endpoint" class="space-y-2">
          <h3 class="text-sm font-medium flex items-center gap-2">
            <Settings class="w-4 h-4" />
            Endpoint Configuration
          </h3>
          <div class="space-y-3 text-xs">
            <div v-if="step.endpoint.url">
              <span class="text-muted-foreground">URL:</span>
              <p class="font-mono mt-1 break-all">{{ step.endpoint.url }}</p>
            </div>
            <div v-if="step.endpoint.method">
              <span class="text-muted-foreground">Method:</span>
              <p class="mt-1 font-medium">{{ step.endpoint.method }}</p>
            </div>
            <div v-if="step.endpoint.interval">
              <span class="text-muted-foreground">Interval:</span>
              <p class="mt-1">{{ step.endpoint.interval }}</p>
            </div>
            <div v-if="step.endpoint.timeout">
              <span class="text-muted-foreground">Timeout:</span>
              <p class="mt-1">{{ step.endpoint.timeout }}</p>
            </div>
          </div>
        </div>

        <!-- Result Errors (separate from step errors) -->
        <div v-if="step.result?.errors?.length" class="space-y-2">
          <h3 class="text-sm font-medium flex items-center gap-2 text-red-600 dark:text-red-400">
            <AlertCircle class="w-4 h-4" />
            Result Errors ({{ step.result.errors.length }})
          </h3>
          <div class="space-y-2 max-h-32 overflow-y-auto">
            <div v-for="(error, index) in step.result.errors" :key="index"
                 class="p-3 bg-red-50 dark:bg-red-900/50 border border-red-200 dark:border-red-700 rounded text-sm font-mono text-red-800 dark:text-red-300 break-all">
              {{ error }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { X, AlertCircle, RotateCcw, Download, CheckCircle, XCircle, SkipForward, Pause, Clock, Settings } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { formatDuration } from '@/utils/format'
import { prettifyTimestamp } from '@/utils/time'

const props = defineProps({
  step: { type: Object, required: true },
  index: { type: Number, required: true }
})

defineEmits(['close'])

const statusIcon = computed(() => {
  switch (props.step.status) {
    case 'success': return CheckCircle
    case 'failed': return XCircle
    case 'skipped': return SkipForward
    case 'not-started': return Pause
    default: return Pause
  }
})

const iconClasses = computed(() => {
  switch (props.step.status) {
    case 'success': return 'text-green-600 dark:text-green-400'
    case 'failed': return 'text-red-600 dark:text-red-400'
    case 'skipped': return 'text-gray-600 dark:text-gray-400'
    default: return 'text-blue-600 dark:text-blue-400'
  }
})

</script>