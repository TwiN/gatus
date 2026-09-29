<template>
  <div class="relative w-full" style="height: 300px;">
    <div v-if="loading" class="absolute inset-0 flex items-center justify-center bg-background/50">
      <Loading />
    </div>
    <div v-else-if="error" class="absolute inset-0 flex items-center justify-center text-muted-foreground">
      {{ error }}
    </div>
    <Line v-else :data="chartData" :options="chartOptions" />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { Line } from 'vue-chartjs'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler, TimeScale } from 'chart.js'
import annotationPlugin from 'chartjs-plugin-annotation'
import 'chartjs-adapter-date-fns'
import { generatePrettyTimeDifference } from '@/utils/time'
import Loading from './Loading.vue'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler, TimeScale, annotationPlugin)

const props = defineProps({
  endpointKey: {
    type: String,
    required: true
  },
  duration: {
    type: String,
    required: true,
    validator: (value) => ['24h', '7d', '30d'].includes(value)
  },
  serverUrl: {
    type: String,
    default: '..'
  },
  events: {
    type: Array,
    default: () => []
  }
})

const loading = ref(true)
const error = ref(null)
const timestamps = ref([])
const values = ref([])
const isDark = ref(document.documentElement.classList.contains('dark'))
const hoveredEventIndex = ref(null)

// Helper function to get color for unhealthy events
const getEventColor = () => {
  // Only UNHEALTHY events are displayed on the chart
  return 'rgba(239, 68, 68, 0.8)' // Red
}

// Filter events based on selected duration and calculate durations
const filteredEvents = computed(() => {
  if (!props.events || props.events.length === 0) {
    return []
  }

  const now = new Date()
  let fromTime
  switch (props.duration) {
    case '24h':
      fromTime = new Date(now.getTime() - 24 * 60 * 60 * 1000)
      break
    case '7d':
      fromTime = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
      break
    case '30d':
      fromTime = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
      break
    default:
      return []
  }

  // Only include UNHEALTHY events and calculate their duration
  const unhealthyEvents = []
  for (let i = 0; i < props.events.length; i++) {
    const event = props.events[i]
    if (event.type !== 'UNHEALTHY') continue

    const eventTime = new Date(event.timestamp)
    if (eventTime < fromTime || eventTime > now) continue

    // Find the next event to calculate duration
    let duration = null
    let isOngoing = false
    if (i + 1 < props.events.length) {
      const nextEvent = props.events[i + 1]
      duration = generatePrettyTimeDifference(nextEvent.timestamp, event.timestamp)
    } else {
      // Still ongoing - calculate duration from event time to now
      duration = generatePrettyTimeDifference(now, event.timestamp)
      isOngoing = true
    }

    unhealthyEvents.push({
      ...event,
      duration,
      isOngoing
    })
  }

  return unhealthyEvents
})

const chartData = computed(() => {
  if (timestamps.value.length === 0) {
    return {
      labels: [],
      datasets: []
    }
  }
  const labels = timestamps.value.map(ts => new Date(ts))
  return {
    labels,
    datasets: [{
      label: 'Response Time (ms)',
      data: values.value,
      borderColor: isDark.value ? 'rgb(96, 165, 250)' : 'rgb(59, 130, 246)',
      backgroundColor: isDark.value ? 'rgba(96, 165, 250, 0.1)' : 'rgba(59, 130, 246, 0.1)',
      borderWidth: 2,
      pointRadius: 2,
      pointHoverRadius: 4,
      tension: 0.1,
      fill: true
    }]
  }
})

const chartOptions = computed(() => {
  // Include hoveredEventIndex in dependency tracking
  // eslint-disable-next-line no-unused-vars
  const _ = hoveredEventIndex.value

  // Calculate max Y value for positioning annotations
  const maxY = values.value.length > 0 ? Math.max(...values.value) : 0
  const midY = maxY / 2

  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index',
      intersect: false
    },
    plugins: {
      legend: {
        display: false
      },
      tooltip: {
        backgroundColor: isDark.value ? 'rgba(31, 41, 55, 0.95)' : 'rgba(255, 255, 255, 0.95)',
        titleColor: isDark.value ? '#f9fafb' : '#111827',
        bodyColor: isDark.value ? '#d1d5db' : '#374151',
        borderColor: isDark.value ? '#4b5563' : '#e5e7eb',
        borderWidth: 1,
        padding: 12,
        displayColors: false,
        callbacks: {
          title: (tooltipItems) => {
            if (tooltipItems.length > 0) {
              const date = new Date(tooltipItems[0].parsed.x)
              return date.toLocaleString()
            }
            return ''
          },
          label: (context) => {
            const value = context.parsed.y
            return `${value}ms`
          }
        }
      },
      annotation: {
        annotations: filteredEvents.value.reduce((acc, event, index) => {
          // Find closest data point to determine annotation position
          const eventTimestamp = new Date(event.timestamp).getTime()
          let closestValue = 0

          if (timestamps.value.length > 0 && values.value.length > 0) {
            const closestIndex = timestamps.value.reduce((closest, ts, idx) => {
              const tsTime = new Date(ts).getTime()
              const currentDistance = Math.abs(tsTime - eventTimestamp)
              const closestDistance = Math.abs(new Date(timestamps.value[closest]).getTime() - eventTimestamp)
              return currentDistance < closestDistance ? idx : closest
            }, 0)
            closestValue = values.value[closestIndex]
          }

          // Position annotation at bottom if data point is in lower half, at top if in upper half
          const position = closestValue <= midY ? 'end' : 'start'

          acc[`event-${index}`] = {
            type: 'line',
            xMin: new Date(event.timestamp),
            xMax: new Date(event.timestamp),
            borderColor: getEventColor(),
            borderWidth: 1,
            borderDash: [5, 5],
            enter() {
              hoveredEventIndex.value = index
            },
            leave() {
              hoveredEventIndex.value = null
            },
            label: {
              display: () => hoveredEventIndex.value === index,
              content: [event.isOngoing ? `Status: ONGOING` : `Status: RESOLVED`, `Unhealthy for ${event.duration}`, `Started at ${new Date(event.timestamp).toLocaleString()}`],
              backgroundColor: getEventColor(),
              color: '#ffffff',
              font: {
                size: 11
              },
              padding: 6,
              position
            }
          }
          return acc
        }, {})
      }
    },
    scales: {
      x: {
        type: 'time',
        time: {
          unit: props.duration === '24h' ? 'hour' : props.duration === '7d' ? 'day' : 'day',
          displayFormats: {
            hour: 'MMM d, ha',
            day: 'MMM d'
          }
        },
        grid: {
          color: isDark.value ? 'rgba(75, 85, 99, 0.3)' : 'rgba(229, 231, 235, 0.8)',
          drawBorder: false
        },
        ticks: {
          color: isDark.value ? '#9ca3af' : '#6b7280',
          maxRotation: 0,
          autoSkipPadding: 20
        }
      },
      y: {
        beginAtZero: true,
        grid: {
          color: isDark.value ? 'rgba(75, 85, 99, 0.3)' : 'rgba(229, 231, 235, 0.8)',
          drawBorder: false
        },
        ticks: {
          color: isDark.value ? '#9ca3af' : '#6b7280',
          callback: (value) => `${value}ms`
        }
      }
    }
  }
})

const fetchData = async () => {
  loading.value = true
  error.value = null
  try {
    const response = await fetch(`${props.serverUrl}/api/v1/endpoints/${props.endpointKey}/response-times/${props.duration}/history`, {
      credentials: 'include'
    })
    if (response.status === 200) {
      const data = await response.json()
      timestamps.value = data.timestamps || []
      values.value = data.values || []
    } else {
      error.value = 'Failed to load chart data'
      console.error('[ResponseTimeChart] Error:', await response.text())
    }
  } catch (err) {
    error.value = 'Failed to load chart data'
    console.error('[ResponseTimeChart] Error:', err)
  } finally {
    loading.value = false
  }
}

watch(() => props.duration, () => {
  fetchData()
})

onMounted(() => {
  fetchData()
  const observer = new MutationObserver(() => {
    isDark.value = document.documentElement.classList.contains('dark')
  })
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  onUnmounted(() => observer.disconnect())
})
</script>