<template>
  <div v-if="announcements && announcements.length" class="announcement-container mb-6">
    <div 
      :class="[
        'rounded-lg border bg-card text-card-foreground shadow-sm transition-all duration-200',
        containerClasses
      ]"
    >
      <!-- Header -->
      <div 
        :class="[
          'announcement-header px-4 py-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors',
          isCollapsed ? 'rounded-lg' : 'rounded-t-lg border-b border-gray-200 dark:border-gray-600'
        ]"
        @click="toggleCollapsed"
      >
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <component :is="mostRecentIcon" :class="['w-5 h-5', mostRecentIconClass]" />
            <h2 class="text-base font-semibold text-gray-900 dark:text-gray-100">Announcements</h2>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              ({{ announcements.length }})
            </span>
          </div>
          <ChevronDown 
            :class="[
              'w-4 h-4 text-gray-500 dark:text-gray-400 transition-transform duration-200',
              isCollapsed ? '-rotate-90' : 'rotate-0'
            ]"
          />
        </div>
      </div>

      <!-- Timeline Content -->
      <div 
        v-if="!isCollapsed"
        class="announcement-content p-4 transition-all duration-200 rounded-b-lg"
      >
        <div class="relative">
          <!-- Announcements -->
          <div class="space-y-3">
            <div
              v-for="(group, date) in groupedAnnouncements"
              :key="date"
              class="relative"
            >
              <!-- Vertical line from date to last icon -->
              <div 
                v-if="group.length > 0"
                class="absolute left-3 w-0.5 bg-gray-300 dark:bg-gray-600 pointer-events-none"
                :style="getTimelineHeight(group)"
              ></div>

              <!-- Date Header -->
              <div class="flex items-center gap-3 mb-2 relative">
                <div class="relative z-10 bg-white dark:bg-gray-800 px-2 py-1 rounded-md border border-gray-200 dark:border-gray-600">
                  <time class="text-xs font-medium text-gray-600 dark:text-gray-300">
                    {{ formatDate(date) }}
                  </time>
                </div>
                <div class="flex-1 border-t border-gray-200 dark:border-gray-600"></div>
              </div>

              <!-- Announcements for this date -->
              <div class="space-y-2 ml-7 relative">
                <div
                  v-for="(announcement, index) in group"
                  :key="`${date}-${index}-${announcement.timestamp}`"
                  class="relative"
                >
                  <!-- Timeline Icon -->
                  <div 
                    :class="[
                      'absolute -left-[26px] top-1/2 -translate-y-1/2 w-5 h-5 rounded-full border bg-white dark:bg-gray-800 flex items-center justify-center z-10',
                      getTypeClasses(announcement.type).border
                    ]"
                  >
                    <component 
                      :is="getTypeIcon(announcement.type)" 
                      :class="['w-3 h-3', getTypeClasses(announcement.type).iconColor]"
                    />
                  </div>

                  <!-- Announcement Card -->
                  <div 
                    :class="[
                      'rounded-md border p-3 transition-all duration-200 hover:shadow-sm',
                      getTypeClasses(announcement.type).background
                    ]"
                  >
                    <div class="flex items-center justify-between gap-3">
                      <div class="flex-1 min-w-0">
                        <p class="text-sm leading-relaxed text-gray-900 dark:text-gray-100">{{ announcement.message }}</p>
                      </div>
                      <time 
                        :class="[
                          'text-xs font-mono whitespace-nowrap',
                          getTypeClasses(announcement.type).text
                        ]"
                        :title="formatFullTimestamp(announcement.timestamp)"
                      >
                        {{ formatTime(announcement.timestamp) }}
                      </time>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { XCircle, AlertTriangle, Info, CheckCircle, Circle, ChevronDown } from 'lucide-vue-next'

// Props
const props = defineProps({
  announcements: {
    type: Array,
    default: () => []
  }
})

// Collapse state
const isCollapsed = ref(false)

// Methods
const toggleCollapsed = () => {
  isCollapsed.value = !isCollapsed.value
}

// Type configurations
const typeConfigs = {
  outage: {
    icon: XCircle,
    background: 'bg-red-50 border-gray-200 dark:bg-red-900/50 dark:border-gray-600',
    border: 'border-red-500',
    iconColor: 'text-red-600 dark:text-red-400',
    text: 'text-red-700 dark:text-red-300'
  },
  warning: {
    icon: AlertTriangle,
    background: 'bg-yellow-50 border-gray-200 dark:bg-yellow-900/50 dark:border-gray-600',
    border: 'border-yellow-500',
    iconColor: 'text-yellow-600 dark:text-yellow-400',
    text: 'text-yellow-700 dark:text-yellow-300'
  },
  information: {
    icon: Info,
    background: 'bg-blue-50 border-gray-200 dark:bg-blue-900/50 dark:border-gray-600',
    border: 'border-blue-500',
    iconColor: 'text-blue-600 dark:text-blue-400',
    text: 'text-blue-700 dark:text-blue-300'
  },
  operational: {
    icon: CheckCircle,
    background: 'bg-green-50 border-gray-200 dark:bg-green-900/50 dark:border-gray-600',
    border: 'border-green-500',
    iconColor: 'text-green-600 dark:text-green-400',
    text: 'text-green-700 dark:text-green-300'
  },
  none: {
    icon: Circle,
    background: 'bg-gray-50 border-gray-200 dark:bg-gray-800/50 dark:border-gray-600',
    border: 'border-gray-500',
    iconColor: 'text-gray-600 dark:text-gray-400',
    text: 'text-gray-700 dark:text-gray-300'
  }
}

// Computed properties
const mostRecentAnnouncement = computed(() => {
  return props.announcements && props.announcements.length > 0 ? props.announcements[0] : null
})

const mostRecentIcon = computed(() => {
  const type = mostRecentAnnouncement.value?.type || 'none'
  return typeConfigs[type]?.icon || Circle
})

const mostRecentIconClass = computed(() => {
  const type = mostRecentAnnouncement.value?.type || 'none'
  return typeConfigs[type]?.iconColor || 'text-gray-600 dark:text-gray-400'
})

const containerClasses = computed(() => {
  const type = mostRecentAnnouncement.value?.type || 'none'
  const config = typeConfigs[type]
  // Add a subtle left border accent to indicate announcement type
  return `border-l-4 ${config.border.replace('border-', 'border-l-')}`
})

const groupedAnnouncements = computed(() => {
  if (!props.announcements || props.announcements.length === 0) {
    return {}
  }

  const groups = {}
  props.announcements.forEach(announcement => {
    const date = new Date(announcement.timestamp).toDateString()
    if (!groups[date]) {
      groups[date] = []
    }
    groups[date].push(announcement)
  })

  return groups
})

// Helper functions
const getTypeIcon = (type) => {
  return typeConfigs[type]?.icon || Circle
}

const getTypeClasses = (type) => {
  return typeConfigs[type] || typeConfigs.none
}

const getTimelineHeight = (group) => {
  const height = group.length === 1 ? '2rem' : `${2 + (group.length - 1) * 3.5}rem`
  return {
    top: '1.5rem',
    height
  }
}

const formatDate = (dateString) => {
  const date = new Date(dateString)
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)

  if (date.toDateString() === today.toDateString()) {
    return 'Today'
  } else if (date.toDateString() === yesterday.toDateString()) {
    return 'Yesterday'
  } else {
    return date.toLocaleDateString('en-US', {
      weekday: 'long',
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    })
  }
}

const formatTime = (timestamp) => {
  return new Date(timestamp).toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  })
}

const formatFullTimestamp = (timestamp) => {
  return new Date(timestamp).toLocaleString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    timeZoneName: 'short'
  })
}
</script>

<style scoped>
.announcement-container {
  animation: slideDown 0.3s ease-out;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Responsive adjustments */
@media (max-width: 640px) {
  .announcement-container .ml-7 {
    margin-left: 1.5rem;
  }
}
</style>