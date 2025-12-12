<template>
  <div id="settings" class="fixed bottom-4 left-4 z-50">
    <div class="flex items-center gap-1 bg-background/95 backdrop-blur-sm border rounded-full shadow-md p-1">
      <!-- Refresh Rate -->
      <button 
        @click="showRefreshMenu = !showRefreshMenu"
        :aria-label="`Refresh interval: ${formatRefreshInterval(refreshIntervalValue)}`"
        :aria-expanded="showRefreshMenu"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-full hover:bg-accent transition-colors relative"
      >
        <RefreshCw class="w-3.5 h-3.5 text-muted-foreground" />
        <span class="text-xs font-medium">{{ formatRefreshInterval(refreshIntervalValue) }}</span>
        
        <!-- Refresh Rate Dropdown -->
        <div 
          v-if="showRefreshMenu"
          @click.stop
          class="absolute bottom-full left-0 mb-2 bg-popover border rounded-lg shadow-lg overflow-hidden"
        >
          <button
            v-for="interval in REFRESH_INTERVALS"
            :key="interval.value"
            @click="selectRefreshInterval(interval.value)"
            :class="[
              'block w-full px-4 py-2 text-xs text-left hover:bg-accent transition-colors',
              refreshIntervalValue === interval.value && 'bg-accent'
            ]"
          >
            {{ interval.label }}
          </button>
        </div>
      </button>

      <!-- Divider -->
      <div class="h-5 w-px bg-border/50" />

      <!-- Theme Toggle -->
      <button
        @click="toggleDarkMode"
        :aria-label="darkMode ? 'Switch to light mode' : 'Switch to dark mode'"
        class="p-1.5 rounded-full hover:bg-accent transition-colors group relative"
      >
        <Sun v-if="darkMode" class="h-3.5 w-3.5 transition-all" />
        <Moon v-else class="h-3.5 w-3.5 transition-all" />
        
        <!-- Tooltip -->
        <div class="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2 py-1 bg-popover text-popover-foreground text-xs rounded-md shadow-md opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none whitespace-nowrap">
          {{ darkMode ? 'Light mode' : 'Dark mode' }}
        </div>
      </button>
    </div>
  </div>
</template>


<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { Sun, Moon, RefreshCw } from 'lucide-vue-next'

const emit = defineEmits(['refreshData'])

// Constants
const REFRESH_INTERVALS = [
  { value: '10', label: '10s' },
  { value: '30', label: '30s' },
  { value: '60', label: '1m' },
  { value: '120', label: '2m' },
  { value: '300', label: '5m' },
  { value: '600', label: '10m' }
]
const DEFAULT_REFRESH_INTERVAL = '300'
const THEME_COOKIE_NAME = 'theme'
const THEME_COOKIE_MAX_AGE = 31536000 // 1 year
const STORAGE_KEYS = {
  REFRESH_INTERVAL: 'gatus:refresh-interval'
}

// Helper functions
function wantsDarkMode() {
  const themeFromCookie = document.cookie.match(new RegExp(`${THEME_COOKIE_NAME}=(dark|light);?`))?.[1]
  return themeFromCookie === 'dark' || (!themeFromCookie && (window.matchMedia('(prefers-color-scheme: dark)').matches || document.documentElement.classList.contains("dark")))
}

function getStoredRefreshInterval() {
  const stored = localStorage.getItem(STORAGE_KEYS.REFRESH_INTERVAL)
  const parsedValue = stored && parseInt(stored)
  const isValid = parsedValue && parsedValue >= 10 && REFRESH_INTERVALS.some(i => i.value === stored)
  return isValid ? stored : DEFAULT_REFRESH_INTERVAL
}

// State
const refreshIntervalValue = ref(getStoredRefreshInterval())
const darkMode = ref(wantsDarkMode())
const showRefreshMenu = ref(false)
let refreshIntervalHandler = null

// Methods
const formatRefreshInterval = (value) => {
  const interval = REFRESH_INTERVALS.find(i => i.value === value)
  return interval ? interval.label : `${value}s`
}

const setRefreshInterval = (seconds) => {
  localStorage.setItem(STORAGE_KEYS.REFRESH_INTERVAL, seconds)
  if (refreshIntervalHandler) {
    clearInterval(refreshIntervalHandler)
  }
  refreshIntervalHandler = setInterval(() => {
    refreshData()
  }, seconds * 1000)
}

const refreshData = () => {
  emit('refreshData')
}

const selectRefreshInterval = (value) => {
  refreshIntervalValue.value = value
  showRefreshMenu.value = false
  refreshData()
  setRefreshInterval(value)
}

// Close menu when clicking outside
const handleClickOutside = (event) => {
  const settings = document.getElementById('settings')
  if (settings && !settings.contains(event.target)) {
    showRefreshMenu.value = false
  }
}

const setThemeCookie = (theme) => {
  document.cookie = `${THEME_COOKIE_NAME}=${theme}; path=/; max-age=${THEME_COOKIE_MAX_AGE}; samesite=strict`
}

const toggleDarkMode = () => {
  const newTheme = wantsDarkMode() ? 'light' : 'dark'
  setThemeCookie(newTheme)
  applyTheme()
}

const applyTheme = () => {
  const isDark = wantsDarkMode()
  darkMode.value = isDark
  document.documentElement.classList.toggle('dark', isDark)
}

// Lifecycle
onMounted(() => {
  setRefreshInterval(refreshIntervalValue.value)
  applyTheme()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  if (refreshIntervalHandler) {
    clearInterval(refreshIntervalHandler)
  }
  document.removeEventListener('click', handleClickOutside)
})
</script>


<style scoped>
/* Animations for smooth transitions */
@keyframes slideIn {
  from {
    transform: translateX(-20px);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

#settings {
  animation: slideIn 0.3s ease-out;
}

#settings > div {
  transition: all 0.2s ease;
}

#settings > div:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1);
}
</style>
