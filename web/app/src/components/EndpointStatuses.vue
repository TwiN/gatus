<template>
  <div class="endpoint-statuses">
    <!-- Filters always visible -->
    <div class="controls flex items-center space-x-4 mb-4">
      <input
        ref="searchInput"
        v-model="search"
        @input="onSearchInput"
        type="text"
        placeholder="Search endpoints..."
        class="rounded-md px-2 py-1 bg-gray-700 text-gray-100 border border-gray-600 focus:outline-none focus:ring-2 focus:ring-green-500"
      />
      <select
        v-model="statusFilter"
        @change="onFilterChange"
        class="rounded-md px-2 py-1 bg-gray-700 text-gray-100 border border-gray-600 focus:outline-none focus:ring-2 focus:ring-green-500"
      >
        <option value="">All</option>
        <option value="success">Success</option>
        <option value="error">Error</option>
      </select>
    </div>

    <!-- Loading spinner -->
    <Loading v-if="loading" class="h-64 w-64 px-4 my-24" />

    <!-- Endpoints and pagination -->
    <div v-else>
      <div v-if="endpointData.results.length > 0">
        <Endpoints
          :endpointStatuses="endpointData.results"
          :showStatusOnHover="true"
          @showTooltip="emitTooltip"
          @toggleShowAverageResponseTime="emitToggle"
          :showAverageResponseTime="showAverage"
        />
      </div>
      <div v-else class="text-gray-400 text-sm mt-4">
        No endpoints found.
      </div>

      <div
        class="pagination flex justify-center space-x-2 mt-4"
        v-if="totalPages > 1"
      >
        <button
          @click="prevPage"
          :disabled="page <= 1"
          class="px-3 py-1 rounded-md bg-gray-700 text-gray-100 disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-600"
        >
          Prev
        </button>
        <span class="px-2 py-1 text-gray-200">
          {{ page }} / {{ totalPages }}
        </span>
        <button
          @click="nextPage"
          :disabled="page >= totalPages"
          class="px-3 py-1 rounded-md bg-gray-700 text-gray-100 disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-600"
        >
          Next
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, watch } from 'vue';
import debounce from 'lodash/debounce';
import Endpoints from '@/components/Endpoints.vue';
import Loading from '@/components/Loading.vue';
import { SERVER_URL } from '@/main';

export default {
  name: 'EndpointStatuses',
  components: { Endpoints, Loading },
  emits: ['showTooltip', 'toggleShowAverageResponseTime'],
  setup(_, { emit }) {
    const page = ref(1);
    const pageSize = 20;
    const totalPages = ref(1);

    // Load saved filters from localStorage if they exist
    const LOCAL_STORAGE_KEY = 'endpoint-filters';
    let savedFilters = {};
    try {
      savedFilters = JSON.parse(localStorage.getItem(LOCAL_STORAGE_KEY)) || {};
    } catch (e) {
      console.error('Failed to parse local storage:', e);
    }

    const search = ref(savedFilters.search || '');
    const statusFilter = ref(savedFilters.statusFilter || '');

    const endpointData = ref({ results: [], total: 0 });
    const loading = ref(false);
    const showAverage = ref(true);

    const saveFilters = () => {
      localStorage.setItem(
        LOCAL_STORAGE_KEY,
        JSON.stringify({
          search: search.value,
          statusFilter: statusFilter.value,
        })
      );
    };


    const fetchPage = async () => {
      loading.value = true;

      const params = new URLSearchParams({
        page: page.value.toString(),
        pageSize: pageSize.toString(),
      });
      if (search.value) params.append('search', search.value);
      if (statusFilter.value) params.append('status', statusFilter.value);

      try {
        const res = await fetch(
          `${SERVER_URL}/api/v1/endpoints/statuses?${params}`,
          { credentials: 'include' }
        );
        const data = await res.json();

        console.log("Fetched data:", data);

        endpointData.value.results = (Array.isArray(data) ? data : []).map(item => {
          if (!item.group) {
            item.group = 'Ungrouped';
          }
          return item;
        });

        endpointData.value.total = res.headers.has('X-Total-Count')
          ? parseInt(res.headers.get('X-Total-Count'))
          : endpointData.value.results.length;

        totalPages.value = Math.max(
          1,
          Math.ceil(endpointData.value.total / pageSize)
        );
      } catch (e) {
        console.error('Failed to fetch endpoints page', e);
      } finally {
        loading.value = false;
      }
    };


    // Debounced search
    const debouncedFetch = debounce(() => {
      page.value = 1;
      fetchPage();
      saveFilters();
    }, 300);

    watch([search, statusFilter], debouncedFetch, { immediate: false });

    watch(page, () => {
      fetchPage();
      saveFilters();
    });

    const onSearchInput = () => {
      debouncedFetch();
    };

    const onFilterChange = () => {
      debouncedFetch();
    };

    const prevPage = () => {
      if (page.value > 1) {
        page.value -= 1;
      }
    };

    const nextPage = () => {
      if (page.value < totalPages.value) {
        page.value += 1;
      }
    };

    const emitTooltip = (result, event) => {
      emit('showTooltip', result, event);
    };

    const emitToggle = () => {
      emit('toggleShowAverageResponseTime');
    };

    // Initial fetch on mount
    fetchPage();

    return {
      page,
      totalPages,
      search,
      statusFilter,
      endpointData,
      loading,
      showAverage,
      onSearchInput,
      onFilterChange,
      prevPage,
      nextPage,
      emitTooltip,
      emitToggle,
    };
  },
};
</script>

<style scoped>
/* No additional styles */
</style>
