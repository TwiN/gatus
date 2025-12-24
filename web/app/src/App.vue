<template>
  <div id="global" class="bg-background text-foreground">
    <!-- Loading State -->
    <div v-if="!retrievedConfig" class="flex items-center justify-center min-h-screen">
      <Loading size="lg" />
    </div>

    <!-- Main App Container -->
    <div v-else-if="!config || !config.oidc || config.authenticated" class="relative">
      <!-- Header -->
      <header class="border-b bg-card/50 backdrop-blur supports-[backdrop-filter]:bg-card/60">
        <div class="container mx-auto px-4 py-4 max-w-7xl">
          <div class="flex items-center justify-between">
            <!-- Logo and Title -->
            <div class="flex items-center gap-4">
              <component 
                :is="link ? 'a' : 'div'" 
                :href="link" 
                target="_blank"
                :class="['flex items-center gap-3', link && 'hover:opacity-80 transition-opacity']"
              >
                <div class="w-12 h-12 flex items-center justify-center">
                  <img 
                    v-if="logo" 
                    :src="logo" 
                    alt="Gatus" 
                    class="w-full h-full object-contain"
                  />
                  <img 
                    v-else 
                    src="./assets/logo.svg" 
                    alt="Gatus" 
                    class="w-full h-full object-contain"
                  />
                </div>
                <div>
                  <h1 class="text-2xl font-bold tracking-tight">{{ header }}</h1>
                  <p v-if="buttons && buttons.length" class="text-sm text-muted-foreground">
                    System Monitoring Dashboard
                  </p>
                </div>
              </component>
            </div>

            <!-- Right Side Actions -->
            <div class="flex items-center gap-2">
              <!-- Navigation Links (Desktop) -->
              <nav v-if="buttons && buttons.length" class="hidden md:flex items-center gap-1">
                <a 
                  v-for="button in buttons" 
                  :key="button.name" 
                  :href="button.link" 
                  target="_blank"
                  class="px-3 py-2 text-sm font-medium rounded-md hover:bg-accent hover:text-accent-foreground transition-colors"
                >
                  {{ button.name }}
                </a>
              </nav>

              <!-- Mobile Menu Button -->
              <Button 
                v-if="buttons && buttons.length" 
                variant="ghost" 
                size="icon" 
                class="md:hidden"
                @click="mobileMenuOpen = !mobileMenuOpen"
              >
                <Menu v-if="!mobileMenuOpen" class="h-5 w-5" />
                <X v-else class="h-5 w-5" />
              </Button>
            </div>
          </div>

          <!-- Mobile Navigation -->
          <nav 
            v-if="buttons && buttons.length && mobileMenuOpen" 
            class="md:hidden mt-4 pt-4 border-t space-y-1"
          >
            <a 
              v-for="button in buttons" 
              :key="button.name" 
              :href="button.link" 
              target="_blank"
              class="block px-3 py-2 text-sm font-medium rounded-md hover:bg-accent hover:text-accent-foreground transition-colors"
              @click="mobileMenuOpen = false"
            >
              {{ button.name }}
            </a>
          </nav>
        </div>
      </header>

      <!-- Main Content -->
      <main class="relative">
        <router-view @showTooltip="showTooltip" :announcements="announcements" />
      </main>

      <!-- Footer -->
      <footer class="border-t mt-auto">
        <div class="container mx-auto px-4 py-6 max-w-7xl">
          <div class="flex flex-col items-center gap-4">
            <div class="text-sm text-muted-foreground text-center">
              Powered by <a href="https://gatus.io" target="_blank" class="font-medium text-emerald-800 hover:text-emerald-600">Gatus</a>
            </div>
            <Social />
          </div>
        </div>
      </footer>
    </div>

    <!-- OIDC Login Screen -->
    <div v-else id="login-container" class="flex items-center justify-center min-h-screen p-4">
      <Card class="w-full max-w-md">
        <CardHeader class="text-center">
          <img 
            src="./assets/logo.svg" 
            alt="Gatus" 
            class="w-20 h-20 mx-auto mb-4"
          />
          <CardTitle class="text-3xl">Gatus</CardTitle>
          <p class="text-muted-foreground mt-2">System Monitoring Dashboard</p>
        </CardHeader>
        <CardContent>
          <div v-if="route && route.query.error" class="mb-6">
            <div class="p-3 rounded-md bg-destructive/10 border border-destructive/20">
              <p class="text-sm text-destructive text-center">
                <span v-if="route.query.error === 'access_denied'">
                  You do not have access to this status page
                </span>
                <span v-else>{{ route.query.error }}</span>
              </p>
            </div>
          </div>
          
          <a
            :href="`/oidc/login`"
            class="inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 bg-primary text-primary-foreground hover:bg-primary/90 h-11 px-8 w-full"
            @click="isOidcLoading = true"
          >
            <Loading v-if="isOidcLoading" size="xs" />
            <template v-else>
              <LogIn class="mr-2 h-4 w-4" />
              Login with OIDC
            </template>
          </a>
        </CardContent>
      </Card>
    </div>

    <!-- Tooltip -->
    <Tooltip :result="tooltip.result" :event="tooltip.event" :isPersistent="tooltipIsPersistent" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { Menu, X, LogIn } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import Social from './components/Social.vue'
import Tooltip from './components/Tooltip.vue'
import Loading from './components/Loading.vue'

const route = useRoute()

// State
const retrievedConfig = ref(false)
const config = ref({ oidc: false, authenticated: true })
const announcements = ref([])
const tooltip = ref({})
const mobileMenuOpen = ref(false)
const isOidcLoading = ref(false)
const tooltipIsPersistent = ref(false)
let configInterval = null

// Computed properties
const logo = computed(() => {
  return window.config && window.config.logo && window.config.logo !== '{{ .UI.Logo }}' ? window.config.logo : ""
})

const header = computed(() => {
  return window.config && window.config.header && window.config.header !== '{{ .UI.Header }}' ? window.config.header : "Gatus"
})

const link = computed(() => {
  return window.config && window.config.link && window.config.link !== '{{ .UI.Link }}' ? window.config.link : null
})

const buttons = computed(() => {
  return window.config && window.config.buttons ? window.config.buttons : []
})

// Methods
const fetchConfig = async () => {
  try {
    const response = await fetch(`/api/v1/config`, { credentials: 'include' })
    if (response.status === 200) {
      const data = await response.json()
      config.value = data
      announcements.value = data.announcements || []
    }
    retrievedConfig.value = true
  } catch (error) {
    console.error('Failed to fetch config:', error)
    retrievedConfig.value = true
  }
}

const showTooltip = (result, event, action = 'hover') => {
  if (action === 'click') {
    if (!result) {
      // Deselecting
      tooltip.value = {}
      tooltipIsPersistent.value = false
    } else {
      // Selecting new data point
      tooltip.value = { result, event }
      tooltipIsPersistent.value = true
    }
  } else if (action === 'hover') {
    // Only update tooltip on hover if not in persistent mode
    if (!tooltipIsPersistent.value) {
      tooltip.value = { result, event }
    }
  }
}

const handleDocumentClick = (event) => {
  // Close persistent tooltip when clicking outside
  if (tooltipIsPersistent.value) {
    const tooltipElement = document.getElementById('tooltip')
    // Check if click is on a data point bar or inside tooltip
    const clickedDataPoint = event.target.closest('.flex-1.h-6, .flex-1.h-8')

    if (tooltipElement && !tooltipElement.contains(event.target) && !clickedDataPoint) {
      tooltip.value = {}
      tooltipIsPersistent.value = false
      // Emit event to clear selections in child components
      window.dispatchEvent(new CustomEvent('clear-data-point-selection'))
    }
  }
}

// Fetch config on mount and set up interval
onMounted(() => {
  fetchConfig()
  // Refresh config every 10 minutes for announcements
  configInterval = setInterval(fetchConfig, 600000)
  // Add click listener for closing persistent tooltips
  document.addEventListener('click', handleDocumentClick)
})

// Clean up interval on unmount
onUnmounted(() => {
  if (configInterval) {
    clearInterval(configInterval)
    configInterval = null
  }
  // Remove click listener
  document.removeEventListener('click', handleDocumentClick)
})
</script>