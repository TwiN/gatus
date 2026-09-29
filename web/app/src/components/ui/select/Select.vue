<template>
  <div ref="selectRef" class="relative" :class="props.class">
    <button
      @click="toggleDropdown"
      @keydown="handleKeyDown"
      :aria-expanded="isOpen"
      :aria-haspopup="true"
      :aria-label="selectedOption.label || props.placeholder"
      class="flex h-9 sm:h-10 w-full items-center justify-between rounded-md border border-input bg-background px-2 sm:px-3 py-1.5 sm:py-2 text-xs sm:text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
    >
      <span class="truncate">{{ selectedOption.label }}</span>
      <ChevronDown class="h-3 w-3 sm:h-4 sm:w-4 opacity-50 flex-shrink-0 ml-1" />
    </button>
    
    <div
      v-if="isOpen"
      role="listbox"
      class="absolute top-full left-0 z-50 mt-1 w-full rounded-md border bg-popover text-popover-foreground shadow-md animate-in fade-in-0 zoom-in-95"
    >
      <div class="p-1">
        <div
          v-for="(option, index) in options"
          :key="option.value"
          @click="selectOption(option)"
          :class="[
            'relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-6 sm:pl-8 pr-2 text-xs sm:text-sm outline-none hover:bg-accent hover:text-accent-foreground',
            index === focusedIndex && 'bg-accent text-accent-foreground'
          ]"
          role="option"
          :aria-selected="modelValue === option.value"
        >
          <span class="absolute left-1.5 sm:left-2 flex h-3.5 w-3.5 items-center justify-center">
            <Check v-if="modelValue === option.value" class="h-3 w-3 sm:h-4 sm:w-4" />
          </span>
          {{ option.label }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ChevronDown, Check } from 'lucide-vue-next'

const props = defineProps({
  modelValue: { type: String, default: '' },
  options: { type: Array, required: true },
  placeholder: { type: String, default: 'Select...' },
  class: { type: String, default: '' }
})

const emit = defineEmits(['update:modelValue'])

const isOpen = ref(false)
const selectRef = ref(null)
const focusedIndex = ref(-1)

const selectedOption = computed(() => {
  return props.options.find(option => option.value === props.modelValue) || { label: props.placeholder, value: '' }
})

const selectOption = (option) => {
  emit('update:modelValue', option.value)
  isOpen.value = false
}

const toggleDropdown = () => {
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    // Set initial focus to selected option or first option
    const selectedIdx = props.options.findIndex(opt => opt.value === props.modelValue)
    focusedIndex.value = selectedIdx >= 0 ? selectedIdx : 0
  } else {
    focusedIndex.value = -1
  }
}

const handleClickOutside = (event) => {
  if (selectRef.value && !selectRef.value.contains(event.target)) {
    isOpen.value = false
    focusedIndex.value = -1
  }
}

const handleKeyDown = (event) => {
  if (!isOpen.value) {
    if (event.key === 'Enter' || event.key === ' ' || event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault()
      toggleDropdown()
    }
    return
  }

  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      focusedIndex.value = Math.min(focusedIndex.value + 1, props.options.length - 1)
      break
    case 'ArrowUp':
      event.preventDefault()
      focusedIndex.value = Math.max(focusedIndex.value - 1, 0)
      break
    case 'Enter':
    case ' ':
      event.preventDefault()
      if (focusedIndex.value >= 0 && focusedIndex.value < props.options.length) {
        selectOption(props.options[focusedIndex.value])
      }
      break
    case 'Escape':
      event.preventDefault()
      isOpen.value = false
      focusedIndex.value = -1
      break
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>