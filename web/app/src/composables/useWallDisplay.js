import { ref, computed, onMounted, onUnmounted } from 'vue'

const STORAGE_KEY = 'gatus:wall-display'

const wallDisplay = ref(getStoredWallDisplay())

function getStoredWallDisplay() {
  if (typeof localStorage === 'undefined') {
    return false
  }
  return localStorage.getItem(STORAGE_KEY) === 'true'
}

// window.config.density is set by the Go-rendered index template (authoritative).
// This whitelist is a client-side fallback for frontend-dev and unexpected values.
function getConfigDensity() {
  const density = typeof window !== 'undefined' && window.config && window.config.density
  return density === 'compact' ? 'compact' : 'comfortable'
}

function syncDocumentClasses() {
  if (typeof document === 'undefined') {
    return
  }
  const root = document.documentElement
  const compact = wallDisplay.value || getConfigDensity() === 'compact'
  root.classList.toggle('density-compact', compact)
  root.classList.toggle('wall-display', wallDisplay.value)
}

function setWallDisplay(value) {
  wallDisplay.value = value
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(STORAGE_KEY, value ? 'true' : 'false')
  }
  syncDocumentClasses()
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('gatus:wall-display-changed', { detail: value }))
  }
}

function toggleWallDisplay() {
  setWallDisplay(!wallDisplay.value)
}

let storageListenerAttached = false

function handleStorageEvent(event) {
  if (event.key === STORAGE_KEY) {
    wallDisplay.value = event.newValue === 'true'
    syncDocumentClasses()
  }
}

if (typeof document !== 'undefined') {
  syncDocumentClasses()
}

export function useWallDisplay() {
  const isWallDisplay = computed(() => wallDisplay.value)
  const isCompact = computed(() => wallDisplay.value || getConfigDensity() === 'compact')

  onMounted(() => {
    syncDocumentClasses()
    if (!storageListenerAttached && typeof window !== 'undefined') {
      window.addEventListener('storage', handleStorageEvent)
      storageListenerAttached = true
    }
  })

  onUnmounted(() => {
    // Shared listener stays attached for app lifetime
  })

  return {
    isWallDisplay,
    isCompact,
    wallDisplay,
    setWallDisplay,
    toggleWallDisplay,
    syncDocumentClasses,
  }
}