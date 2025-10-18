<template>
  <div v-if="isOpen" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm" @click.self="closeModal">
    <Card class="w-full max-w-2xl max-h-[90vh] flex flex-col">
      <CardHeader class="flex-shrink-0">
        <div class="flex items-center justify-between">
          <CardTitle>API Key Management</CardTitle>
          <button @click="closeModal" class="p-1 rounded-md hover:bg-accent transition-colors" aria-label="Close">
            <X class="h-5 w-5" />
          </button>
        </div>
        <p class="text-sm text-muted-foreground mt-2">
          Generate and manage API keys for programmatic access to Gatus
        </p>
      </CardHeader>

      <CardContent class="flex-1 overflow-y-auto">
        <!-- Loading State -->
        <div v-if="loading" class="flex items-center justify-center py-8">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>

        <!-- Error State -->
        <div v-else-if="error" class="p-4 rounded-md bg-destructive/10 border border-destructive/20">
          <p class="text-sm text-destructive">{{ error }}</p>
        </div>

        <!-- Success: New Key Created -->
        <div v-else-if="newlyCreatedKey" class="space-y-4 mb-6">
          <div class="p-4 rounded-md bg-green-500/10 border border-green-500/20">
            <div class="flex items-start gap-2">
              <AlertCircle class="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
              <div class="flex-1">
                <h3 class="text-sm font-semibold text-green-600 mb-1">API Key Created Successfully</h3>
                <p class="text-xs text-muted-foreground mb-3">
                  Make sure to copy your API key now. You won't be able to see it again!
                </p>
                <div class="flex items-center gap-2">
                  <code class="flex-1 px-3 py-2 text-xs bg-background border rounded-md font-mono break-all">
                    {{ newlyCreatedKey }}
                  </code>
                  <Button @click="copyToClipboard(newlyCreatedKey)" size="sm" variant="outline">
                    <Copy class="h-4 w-4 mr-1" />
                    {{ copiedKey === newlyCreatedKey ? 'Copied!' : 'Copy' }}
                  </Button>
                </div>
              </div>
            </div>
          </div>
          <Button @click="newlyCreatedKey = null" variant="outline" class="w-full">
            <CheckCircle class="h-4 w-4 mr-2" />
            I've saved my key
          </Button>
        </div>

        <!-- Create New Key Form -->
        <div v-else-if="showCreateForm" class="space-y-4 mb-6">
          <div>
            <label for="keyName" class="block text-sm font-medium mb-2">API Key Name</label>
            <input
              id="keyName"
              v-model="newKeyName"
              type="text"
              placeholder="e.g., Production Server, CI/CD Pipeline"
              class="w-full px-3 py-2 text-sm border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-ring"
              @keyup.enter="createAPIKey"
            />
          </div>
          <div class="flex gap-2">
            <Button @click="createAPIKey" :disabled="!newKeyName.trim() || creating" class="flex-1">
              <Key class="h-4 w-4 mr-2" />
              {{ creating ? 'Creating...' : 'Generate API Key' }}
            </Button>
            <Button @click="cancelCreate" variant="outline">
              Cancel
            </Button>
          </div>
        </div>

        <!-- Create Button (when not showing form) -->
        <div v-else-if="!loading && !error" class="mb-6">
          <Button @click="showCreateForm = true" class="w-full">
            <Plus class="h-4 w-4 mr-2" />
            Create New API Key
          </Button>
        </div>

        <!-- API Keys List -->
        <div v-if="!loading && !error && apiKeys.length > 0" class="space-y-3">
          <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wider">Your API Keys</h3>
          <div class="space-y-2">
            <div
              v-for="key in apiKeys"
              :key="key.id"
              class="p-3 rounded-md border bg-card hover:bg-accent/50 transition-colors"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-2">
                    <Key class="h-4 w-4 text-muted-foreground flex-shrink-0" />
                    <h4 class="text-sm font-medium truncate">{{ key.name }}</h4>
                  </div>
                  <div class="mt-1 space-y-0.5 text-xs text-muted-foreground">
                    <p>Created: {{ formatDate(key.created_at) }}</p>
                    <p v-if="key.last_used_at">Last used: {{ formatDate(key.last_used_at) }}</p>
                    <p v-else class="text-yellow-600">Never used</p>
                  </div>
                </div>
                <Button
                  @click="confirmDelete(key)"
                  variant="ghost"
                  size="sm"
                  class="text-destructive hover:text-destructive hover:bg-destructive/10"
                >
                  <Trash2 class="h-4 w-4" />
                </Button>
              </div>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-else-if="!loading && !error && apiKeys.length === 0 && !showCreateForm && !newlyCreatedKey" class="text-center py-8">
          <Key class="h-12 w-12 mx-auto text-muted-foreground mb-3 opacity-50" />
          <p class="text-sm text-muted-foreground">No API keys yet</p>
          <p class="text-xs text-muted-foreground mt-1">Create your first API key to get started</p>
        </div>
      </CardContent>
    </Card>

    <!-- Delete Confirmation Modal -->
    <div v-if="keyToDelete" class="absolute inset-0 flex items-center justify-center p-4 bg-black/50" @click.self="keyToDelete = null">
      <Card class="w-full max-w-md">
        <CardHeader>
          <CardTitle class="text-destructive">Delete API Key</CardTitle>
        </CardHeader>
        <CardContent class="space-y-4">
          <p class="text-sm">
            Are you sure you want to delete the API key <strong>"{{ keyToDelete.name }}"</strong>?
          </p>
          <p class="text-xs text-muted-foreground">
            This action cannot be undone. Any applications using this key will immediately lose access.
          </p>
          <div class="flex gap-2">
            <Button @click="deleteAPIKey" variant="destructive" :disabled="deleting" class="flex-1">
              {{ deleting ? 'Deleting...' : 'Delete' }}
            </Button>
            <Button @click="keyToDelete = null" variant="outline">
              Cancel
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { Key, Plus, Copy, Trash2, X, CheckCircle, AlertCircle } from 'lucide-vue-next'
import Card from './ui/card/Card.vue'
import CardHeader from './ui/card/CardHeader.vue'
import CardTitle from './ui/card/CardTitle.vue'
import CardContent from './ui/card/CardContent.vue'
import Button from './ui/button/Button.vue'

const props = defineProps({
  isOpen: {
    type: Boolean,
    required: true
  },
  baseUrl: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['close'])

const apiKeys = ref([])
const loading = ref(false)
const error = ref(null)
const showCreateForm = ref(false)
const newKeyName = ref('')
const creating = ref(false)
const newlyCreatedKey = ref(null)
const copiedKey = ref(null)
const keyToDelete = ref(null)
const deleting = ref(false)

// Watch for modal open to fetch keys
watch(() => props.isOpen, (isOpen) => {
  if (isOpen) {
    fetchAPIKeys()
    // Reset state
    showCreateForm.value = false
    newKeyName.value = ''
    newlyCreatedKey.value = null
    error.value = null
  }
})

const fetchAPIKeys = async () => {
  loading.value = true
  error.value = null
  try {
    const response = await fetch(`${props.baseUrl}/api/v1/apikeys`, {
      credentials: 'include'
    })
    if (response.ok) {
      const data = await response.json()
      apiKeys.value = data
    } else {
      throw new Error(`Failed to fetch API keys: ${response.status}`)
    }
  } catch (err) {
    console.error('Error fetching API keys:', err)
    error.value = 'Failed to load API keys. Please try again.'
  } finally {
    loading.value = false
  }
}

const createAPIKey = async () => {
  if (!newKeyName.value.trim()) return

  creating.value = true
  error.value = null
  try {
    const response = await fetch(`${props.baseUrl}/api/v1/apikeys`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      credentials: 'include',
      body: JSON.stringify({ name: newKeyName.value.trim() })
    })

    if (response.ok) {
      const data = await response.json()
      newlyCreatedKey.value = data.token
      showCreateForm.value = false
      newKeyName.value = ''
      // Refresh the list
      await fetchAPIKeys()
    } else {
      throw new Error('Failed to create API key')
    }
  } catch (err) {
    console.error('Error creating API key:', err)
    error.value = 'Failed to create API key. Please try again.'
  } finally {
    creating.value = false
  }
}

const cancelCreate = () => {
  showCreateForm.value = false
  newKeyName.value = ''
}

const confirmDelete = (key) => {
  keyToDelete.value = key
}

const deleteAPIKey = async () => {
  if (!keyToDelete.value) return

  deleting.value = true
  error.value = null
  try {
    const response = await fetch(`${props.baseUrl}/api/v1/apikeys/${keyToDelete.value.id}`, {
      method: 'DELETE',
      credentials: 'include'
    })

    if (response.ok || response.status === 204) {
      keyToDelete.value = null
      // Refresh the list
      await fetchAPIKeys()
    } else {
      throw new Error('Failed to delete API key')
    }
  } catch (err) {
    console.error('Error deleting API key:', err)
    error.value = 'Failed to delete API key. Please try again.'
    keyToDelete.value = null
  } finally {
    deleting.value = false
  }
}

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    copiedKey.value = text
    setTimeout(() => {
      copiedKey.value = null
    }, 2000)
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}

const formatDate = (dateString) => {
  const date = new Date(dateString)
  const now = new Date()
  const diffInSeconds = Math.floor((now - date) / 1000)

  if (diffInSeconds < 60) return 'just now'
  if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}m ago`
  if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}h ago`
  if (diffInSeconds < 604800) return `${Math.floor(diffInSeconds / 86400)}d ago`

  return date.toLocaleDateString()
}

const closeModal = () => {
  if (!newlyCreatedKey.value) {
    emit('close')
  }
}
</script>

<style scoped>
/* Additional styling if needed */
</style>
