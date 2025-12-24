<template>
  <div class="flex items-center justify-between">
    <Button
      variant="outline"
      size="sm"
      :disabled="currentPage >= maxPages"
      @click="previousPage"
      class="flex items-center gap-1"
    >
      <ChevronLeft class="h-4 w-4" />
      Previous
    </Button>
    
    <span class="text-sm text-muted-foreground">
      Page {{ currentPage }} of {{ maxPages }}
    </span>
    
    <Button
      variant="outline"
      size="sm"
      :disabled="currentPage <= 1"
      @click="nextPage"
      class="flex items-center gap-1"
    >
      Next
      <ChevronRight class="h-4 w-4" />
    </Button>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'

const props = defineProps({
  numberOfResultsPerPage: Number,
  currentPageProp: {
    type: Number,
    default: 1
  }
})

const emit = defineEmits(['page'])

const currentPage = ref(props.currentPageProp)

const maxPages = computed(() => {
  // Use maximumNumberOfResults from config if available, otherwise default to 100
  let maxResults = 100 // Default value
  // Check if window.config exists and has maximumNumberOfResults
  if (typeof window !== 'undefined' && window.config && window.config.maximumNumberOfResults) {
    const parsed = parseInt(window.config.maximumNumberOfResults)
    if (!isNaN(parsed)) {
      maxResults = parsed
    }
  }
  return Math.ceil(maxResults / props.numberOfResultsPerPage)
})

const nextPage = () => {
  // "Next" should show newer data (lower page numbers)
  currentPage.value--
  emit('page', currentPage.value)
}

const previousPage = () => {
  // "Previous" should show older data (higher page numbers)
  currentPage.value++
  emit('page', currentPage.value)
}
</script>