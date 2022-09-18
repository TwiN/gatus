<template>
  <div class="mt-3 flex text-6xl">
    <div class="flex-1" id="left-page-arrow">
      <button v-if="currentPage < 5" @click="nextPage" class="bg-gray-100 hover:bg-gray-200 text-gray-500 border border-gray-200 px-2 rounded font-mono dark:bg-gray-700 dark:text-gray-200 dark:border-gray-500 dark:hover:bg-gray-600">&lt;</button>
    </div>
    <div class="flex-1 text-right" id="right-page-arrow">
      <button v-if="currentPage > 1" @click="previousPage" class="bg-gray-100 hover:bg-gray-200 text-gray-500 border border-gray-200 px-2 rounded font-mono dark:bg-gray-700 dark:text-gray-200 dark:border-gray-500 dark:hover:bg-gray-600">&gt;</button>
    </div>
  </div>
</template>


<script>
export default {
  name: 'Pagination',
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
  data() {
    return {
      currentPage: 1,
    }
  }
}
</script>

<style>
#left-page-arrow {
  position: fixed;
  left: 10px;
  bottom: 50%;
}

#right-page-arrow {
  position: fixed;
  right: 10px;
  bottom: 50%
}
</style>
