<template>
  <div class="mt-3 flex">
    <div class="flex-1">
      <button v-if="currentPage < 5" @click="nextPage" class="bg-gray-100 hover:bg-gray-200 px-2 rounded border-gray-200 border text-monospace text-gray-500">&lt;</button>
    </div>
    <div class="flex-1 text-right">
      <button v-if="currentPage > 1" @click="previousPage" class="bg-gray-100 hover:bg-gray-200 px-2 rounded border-gray-200 border text-monospace text-gray-500">&gt;</button>
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