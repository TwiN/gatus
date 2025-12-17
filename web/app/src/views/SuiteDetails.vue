<template>
  <div class="suite-details-container bg-background min-h-screen">
    <div class="container mx-auto px-4 py-8 max-w-7xl">
      <!-- Back button and header -->
      <div class="mb-6">
        <Button variant="ghost" size="sm" @click="goBack" class="mb-4">
          <ArrowLeft class="h-4 w-4 mr-2" />
          Back to Dashboard
        </Button>
        
        <div class="flex items-start justify-between">
          <div>
            <h1 class="text-3xl font-bold tracking-tight">{{ suite?.name || 'Loading...' }}</h1>
            <p class="text-muted-foreground mt-2">
              <span v-if="suite?.group">{{ suite.group }} • </span>
              <span v-if="latestResult">
                {{ selectedResult && selectedResult.timestamp !== sortedResults[0]?.timestamp ? 'Ran' : 'Last run' }} {{ formatRelativeTime(latestResult.timestamp) }}
              </span>
            </p>
          </div>
          <div class="flex items-center gap-2">
            <StatusBadge v-if="latestResult" :status="latestResult.success ? 'healthy' : 'unhealthy'" />
            <Button variant="ghost" size="icon" @click="refreshData" title="Refresh">
              <RefreshCw class="h-5 w-5" />
            </Button>
          </div>
        </div>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-20">
        <Loading size="lg" />
      </div>

      <div v-else-if="!suite" class="text-center py-20">
        <AlertCircle class="h-12 w-12 text-muted-foreground mx-auto mb-4" />
        <h3 class="text-lg font-semibold mb-2">Suite not found</h3>
        <p class="text-muted-foreground">The requested suite could not be found.</p>
      </div>

      <div v-else class="space-y-6">
        <!-- Latest Execution -->
        <Card v-if="latestResult">
          <CardHeader>
            <CardTitle>{{ selectedResult?.timestamp === sortedResults[0]?.timestamp ? 'Latest Execution' : `Execution at ${formatTimestamp(selectedResult.timestamp)}` }}</CardTitle>
          </CardHeader>
          <CardContent>
            <div class="space-y-4">
              <!-- Execution stats -->
              <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div>
                  <p class="text-sm text-muted-foreground">Status</p>
                  <p class="text-lg font-medium">{{ latestResult.success ? 'Success' : 'Failed' }}</p>
                </div>
                <div>
                  <p class="text-sm text-muted-foreground">Duration</p>
                  <p class="text-lg font-medium">{{ formatDuration(latestResult.duration) }}</p>
                </div>
                <div>
                  <p class="text-sm text-muted-foreground">Endpoints</p>
                  <p class="text-lg font-medium">{{ latestResult.endpointResults?.length || 0 }}</p>
                </div>
                <div>
                  <p class="text-sm text-muted-foreground">Success Rate</p>
                  <p class="text-lg font-medium">{{ calculateSuccessRate(latestResult) }}%</p>
                </div>
              </div>

              <!-- Enhanced Execution Flow -->
              <div class="mt-6">
                <h3 class="text-lg font-semibold mb-4">Execution Flow</h3>
                <SequentialFlowDiagram
                  :flow-steps="flowSteps"
                  :progress-percentage="executionProgress"
                  :completed-steps="completedStepsCount"
                  :total-steps="flowSteps.length"
                  @step-selected="onStepSelected"
                />
              </div>


              <!-- Errors -->
              <div v-if="latestResult.errors && latestResult.errors.length > 0" class="mt-6">
                <h3 class="text-lg font-semibold mb-3 text-red-500">Suite Errors</h3>
                <div class="space-y-2">
                  <div
                    v-for="(error, index) in latestResult.errors"
                    :key="index"
                    class="bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-300 p-3 rounded-md text-sm"
                  >
                    {{ error }}
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <!-- Execution History -->
        <Card>
          <CardHeader>
            <CardTitle>Execution History</CardTitle>
          </CardHeader>
          <CardContent>
            <div v-if="sortedResults.length > 0" class="space-y-2">
              <div
                v-for="(result, index) in sortedResults"
                :key="index"
                class="flex items-center justify-between p-3 border rounded-lg hover:bg-accent/50 transition-colors cursor-pointer"
                @click="selectedResult = result"
                :class="{ 'bg-accent': selectedResult && selectedResult.timestamp === result.timestamp }"
              >
                <div class="flex items-center gap-3">
                  <StatusBadge :status="result.success ? 'healthy' : 'unhealthy'" size="sm" />
                  <div>
                    <p class="text-sm font-medium">{{ formatTimestamp(result.timestamp) }}</p>
                    <p class="text-xs text-muted-foreground">
                      {{ result.endpointResults?.length || 0 }} endpoints • {{ formatDuration(result.duration) }}
                    </p>
                  </div>
                </div>
                <ChevronRight class="h-4 w-4 text-muted-foreground" />
              </div>
            </div>
            <div v-else class="text-center py-8 text-muted-foreground">
              No execution history available
            </div>
          </CardContent>
        </Card>
      </div>
    </div>

    <Settings @refreshData="fetchData" />
    
    <!-- Step Details Modal -->
    <StepDetailsModal
      v-if="selectedStep"
      :step="selectedStep"
      :index="selectedStepIndex"
      @close="selectedStep = null"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ArrowLeft, RefreshCw, AlertCircle, ChevronRight } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import StatusBadge from '@/components/StatusBadge.vue'
import SequentialFlowDiagram from '@/components/SequentialFlowDiagram.vue'
import StepDetailsModal from '@/components/StepDetailsModal.vue'
import Settings from '@/components/Settings.vue'
import Loading from '@/components/Loading.vue'
import { generatePrettyTimeAgo } from '@/utils/time'
import { formatDuration } from '@/utils/format'

const router = useRouter()
const route = useRoute()

// State
const loading = ref(false)
const suite = ref(null)
const selectedResult = ref(null)
const selectedStep = ref(null)
const selectedStepIndex = ref(0)

// Computed properties
const sortedResults = computed(() => {
  if (!suite.value || !suite.value.results || suite.value.results.length === 0) {
    return []
  }
  // Sort results by timestamp in descending order (most recent first)
  return [...suite.value.results].sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
})

const latestResult = computed(() => {
  if (!suite.value || !suite.value.results || suite.value.results.length === 0) {
    return null
  }
  return selectedResult.value || sortedResults.value[0]
})

// Methods
const fetchData = async () => {
  // Don't show loading state on refresh to prevent UI flicker
  const isInitialLoad = !suite.value
  if (isInitialLoad) {
    loading.value = true
  }

  try {
    const response = await fetch(`/api/v1/suites/${route.params.key}/statuses`, {
      credentials: 'include'
    })

    if (response.status === 200) {
      const data = await response.json()
      const oldSuite = suite.value
      suite.value = data
      if (data.results && data.results.length > 0) {
        // Sort results by timestamp to get the most recent one
        const sorted = [...data.results].sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
        // Update selectedResult if: no result selected, or currently viewing the latest result
        const wasViewingLatest = !selectedResult.value ||
          (oldSuite?.results && selectedResult.value.timestamp === [...oldSuite.results].sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))[0]?.timestamp)
        if (wasViewingLatest) {
          selectedResult.value = sorted[0]
        }
      }
    } else if (response.status === 404) {
      suite.value = null
    } else {
      console.error('[SuiteDetails][fetchData] Error:', await response.text())
    }
  } catch (error) {
    console.error('[SuiteDetails][fetchData] Error:', error)
  } finally {
    if (isInitialLoad) {
      loading.value = false
    }
  }
}

const refreshData = () => {
  fetchData()
}

const goBack = () => {
  router.push('/')
}

const formatRelativeTime = (timestamp) => {
  return generatePrettyTimeAgo(timestamp)
}

const formatTimestamp = (timestamp) => {
  const date = new Date(timestamp)
  return date.toLocaleString()
}

const calculateSuccessRate = (result) => {
  if (!result || !result.endpointResults || result.endpointResults.length === 0) {
    return 0
  }
  
  const successful = result.endpointResults.filter(e => e.success).length
  return Math.round((successful / result.endpointResults.length) * 100)
}

// Flow diagram computed properties
const flowSteps = computed(() => {
  if (!latestResult.value || !latestResult.value.endpointResults) {
    return []
  }
  const results = latestResult.value.endpointResults
  return results.map((result, index) => {
    const endpoint = suite.value?.endpoints?.[index]
    const nextResult = results[index + 1]
    // Determine if this is an always-run endpoint by checking execution pattern
    // If a previous step failed but this one still executed, it must be always-run
    let isAlwaysRun = false
    for (let i = 0; i < index; i++) {
      if (!results[i].success) {
        // A previous step failed, but we're still executing, so this must be always-run
        isAlwaysRun = true
        break
      }
    }
    return {
      name: endpoint?.name || result.name || `Step ${index + 1}`,
      endpoint: endpoint,
      result: result,
      status: determineStepStatus(result, endpoint),
      duration: result.duration || 0,
      isAlwaysRun: isAlwaysRun,
      errors: result.errors || [],
      nextStepStatus: nextResult ? determineStepStatus(nextResult, suite.value?.endpoints?.[index + 1]) : null
    }
  })
})

const completedStepsCount = computed(() => {
  return flowSteps.value.filter(step => step.status === 'success').length
})

const executionProgress = computed(() => {
  if (!flowSteps.value.length) return 0
  return Math.round((completedStepsCount.value / flowSteps.value.length) * 100)
})


// Helper functions
const determineStepStatus = (result) => {
  if (!result) return 'not-started'
  // Check if step was skipped
  if (result.conditionResults && result.conditionResults.some(c => c.condition.includes('SKIP'))) {
    return 'skipped'
  }
  // Check if step failed but is always-run (still shows as failed but executed)
  if (!result.success) {
    return 'failed'
  }
  return 'success'
}


// Event handlers
const onStepSelected = (step, index) => {
  selectedStep.value = step
  selectedStepIndex.value = index
}

// Lifecycle
onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.suite-details-container {
  min-height: 100vh;
}
</style>