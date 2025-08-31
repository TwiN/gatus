<template>
  <div 
    id="tooltip" 
    ref="tooltip" 
    :class="[
      'fixed z-50 px-3 py-2 text-sm rounded-md shadow-lg border transition-all duration-200',
      'bg-popover text-popover-foreground border-border',
      hidden ? 'invisible opacity-0' : 'visible opacity-100'
    ]" 
    :style="`top: ${top}px; left: ${left}px;`"
  >
    <div v-if="result" class="space-y-2">
      <!-- Timestamp -->
      <div>
        <div class="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Timestamp</div>
        <div class="font-mono text-xs">{{ prettifyTimestamp(result.timestamp) }}</div>
      </div>
      
      <!-- Response Time -->
      <div>
        <div class="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Response Time</div>
        <div class="font-mono text-xs">{{ (result.duration / 1000000).toFixed(0) }}ms</div>
      </div>
      
      <!-- Conditions -->
      <div v-if="result.conditionResults && result.conditionResults.length">
        <div class="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Conditions</div>
        <div class="font-mono text-xs space-y-0.5">
          <div 
            v-for="(conditionResult, index) in result.conditionResults" 
            :key="index"
            class="flex items-start gap-1"
          >
            <span :class="conditionResult.success ? 'text-green-500' : 'text-red-500'">
              {{ conditionResult.success ? '✓' : '✗' }}
            </span>
            <span class="break-all">{{ conditionResult.condition }}</span>
          </div>
        </div>
      </div>
      
      <!-- Errors -->
      <div v-if="result.errors && result.errors.length">
        <div class="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Errors</div>
        <div class="font-mono text-xs space-y-0.5">
          <div v-for="(error, index) in result.errors" :key="index" class="text-red-500">
            • {{ error }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
/* eslint-disable no-undef */
import { ref, watch, nextTick } from 'vue'
import { helper } from '@/mixins/helper'

const props = defineProps({
  event: {
    type: [Event, Object],
    default: null
  },
  result: {
    type: Object,
    default: null
  },
  isPersistent: {
    type: Boolean,
    default: false
  }
})

// State
const hidden = ref(true)
const top = ref(0)
const left = ref(0)
const tooltip = ref(null)

// Methods from helper mixin
const { prettifyTimestamp } = helper.methods

const reposition = async () => {
  if (!props.event || !props.event.type) return
  
  await nextTick()
  
  if ((props.event.type === 'mouseenter' || props.event.type === 'click') && tooltip.value) {
    const target = props.event.target
    const targetRect = target.getBoundingClientRect()
    
    // First, position tooltip to get its dimensions
    hidden.value = false
    await nextTick()
    
    const tooltipRect = tooltip.value.getBoundingClientRect()
    
    // Since tooltip uses position: fixed, we work with viewport coordinates
    // getBoundingClientRect() already gives us viewport-relative positions
    
    // Default position: below the target
    let newTop = targetRect.bottom + 8
    let newLeft = targetRect.left
    
    // Check if tooltip would overflow the viewport bottom
    const spaceBelow = window.innerHeight - targetRect.bottom
    const spaceAbove = targetRect.top
    
    if (spaceBelow < tooltipRect.height + 20) {
      // Not enough space below, try above
      if (spaceAbove > tooltipRect.height + 20) {
        // Position above
        newTop = targetRect.top - tooltipRect.height - 8
      } else {
        // Not enough space above either, position at the best spot
        if (spaceAbove > spaceBelow) {
          // More space above
          newTop = 10
        } else {
          // More space below or equal, keep below but adjust
          newTop = window.innerHeight - tooltipRect.height - 10
        }
      }
    }
    
    // Adjust horizontal position if tooltip would overflow right edge
    const spaceRight = window.innerWidth - targetRect.left
    if (spaceRight < tooltipRect.width + 20) {
      // Align right edge of tooltip with right edge of target
      newLeft = targetRect.right - tooltipRect.width
      // Make sure it doesn't go off the left edge
      if (newLeft < 10) {
        newLeft = 10
      }
    }
    
    top.value = Math.round(newTop)
    left.value = Math.round(newLeft)
  } else if (props.event.type === 'mouseleave') {
    // Only hide on mouseleave if not in persistent mode
    if (!props.isPersistent) {
      hidden.value = true
    }
  }
}

// Watchers
watch(() => props.event, (newEvent) => {
  if (newEvent && newEvent.type) {
    if (newEvent.type === 'mouseenter' || newEvent.type === 'click') {
      hidden.value = false
      nextTick(() => reposition())
    } else if (newEvent.type === 'mouseleave') {
      // Only hide on mouseleave if not in persistent mode
      if (!props.isPersistent) {
        hidden.value = true
      }
    }
  }
}, { immediate: true })

watch(() => props.result, () => {
  if (!hidden.value) {
    nextTick(() => reposition())
  }
})

// Watch for persistent state changes and result changes
watch(() => [props.isPersistent, props.result], ([isPersistent, result]) => {
  if (!isPersistent && !result) {
    // Hide tooltip when both persistent mode is off and no result
    hidden.value = true
  } else if (result && (isPersistent || props.event?.type === 'mouseenter')) {
    // Show tooltip when there's a result and either persistent or hovering
    hidden.value = false
    nextTick(() => reposition())
  }
})
</script>