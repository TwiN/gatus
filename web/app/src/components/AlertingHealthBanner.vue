<template>
  <div
    v-if="snapshot"
    class="mb-6 rounded-lg border border-gray-200 dark:border-gray-600 border-l-4 border-l-yellow-500 bg-yellow-50 dark:bg-yellow-900/50 shadow-sm"
  >
    <div class="px-4 py-3 border-b border-gray-200 dark:border-gray-600">
      <div class="flex items-center gap-2">
        <AlertTriangle class="w-5 h-5 text-yellow-600 dark:text-yellow-400 shrink-0" />
        <h2 class="text-base font-semibold text-yellow-800 dark:text-yellow-200">Alerting health</h2>
      </div>
      <p class="text-sm text-yellow-700 dark:text-yellow-300 mt-1">
        Some alerting configuration or delivery issues were detected. Notifications may not be delivered.
      </p>
    </div>
    <div class="px-4 py-3 space-y-3 text-yellow-800 dark:text-yellow-200">
      <div v-if="problemProviders.length" class="space-y-1">
        <h3 class="text-sm font-medium">Providers</h3>
        <ul class="text-sm text-yellow-700 dark:text-yellow-300 space-y-1 list-disc list-inside">
          <li v-for="provider in problemProviders" :key="provider.type">
            <span class="font-mono text-yellow-900 dark:text-yellow-100">{{ provider.type }}</span>
            <span v-if="provider.reason"> — {{ provider.reason }}</span>
          </li>
        </ul>
      </div>
      <div v-if="snapshot.warnings && snapshot.warnings.length" class="space-y-1">
        <h3 class="text-sm font-medium">Configuration</h3>
        <ul class="text-sm text-yellow-700 dark:text-yellow-300 space-y-1 list-disc list-inside">
          <li v-for="(warning, index) in snapshot.warnings" :key="`warning-${index}`">
            {{ warning }}
          </li>
        </ul>
      </div>
      <div v-if="snapshot.delivery && snapshot.delivery.length" class="space-y-1">
        <h3 class="text-sm font-medium">Recent delivery failures</h3>
        <ul class="text-sm text-yellow-700 dark:text-yellow-300 space-y-2">
          <li
            v-for="item in snapshot.delivery"
            :key="item.type"
            class="border-l-2 border-yellow-500 dark:border-yellow-400 pl-3"
          >
            <span class="font-mono text-yellow-900 dark:text-yellow-100">{{ item.type }}</span>
            <span v-if="item.failureCount"> ({{ item.failureCount }} failure<span v-if="item.failureCount !== 1">s</span>)</span>
            <p v-if="item.lastError" class="text-xs text-yellow-600 dark:text-yellow-400 mt-0.5 opacity-90">
              {{ item.lastError }}
            </p>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { AlertTriangle } from 'lucide-vue-next'

const props = defineProps({
  snapshot: {
    type: Object,
    required: true
  }
})

const problemProviders = computed(() => {
  if (!props.snapshot?.providers) {
    return []
  }
  return props.snapshot.providers.filter(p => p.status && p.status !== 'ok')
})
</script>