<template>
  <div class="mt-3 flex">
    <div class="flex-1">
      <button v-if="currentPage < maxPages" @click="nextPage" class="bg-gray-100 hover:bg-gray-200 text-gray-500 border border-gray-200 px-2 rounded font-mono dark:bg-gray-700 dark:text-gray-200 dark:border-gray-500 dark:hover:bg-gray-600">&lt;</button>
    </div>
    <div class="flex-1 text-right">
      <button v-if="currentPage > 1" @click="previousPage" class="bg-gray-100 hover:bg-gray-200 text-gray-500 border border-gray-200 px-2 rounded font-mono dark:bg-gray-700 dark:text-gray-200 dark:border-gray-500 dark:hover:bg-gray-600">&gt;</button>
    </div>
  </div>
</template>


<script>
export default {
  name: 'Pagination',
  props: {
    numberOfResultsPerPage: Number,
  },
  components: {},
  emits: ['page'],
  methods: {
    nextPage() {
      this.currentPage++;
      this.$emit('page', this.currentPage);
    },
    previousPage() {
      this.currentPage--;
      this.$emit('page', this.currentPage);
    }
  },
  computed: {
    maxPages() {
      return Math.ceil(parseInt(window.config.maximumNumberOfResults) / this.numberOfResultsPerPage)
    }
  },
  data() {
    return {
      currentPage: 1,
    }
  }
}
</script>