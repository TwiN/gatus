<script>
import { defineComponent } from 'vue'
import { Line } from 'vue3-chart-v2'

export default defineComponent({
  name: 'Chart',
  extends: Line,
  props: {
    data: {
      type: Object,
      required: true
    }
  },
  watch: {
    data: function (value) {
      this.renderChart(value, this.options)
    }
  },
  data() {
    return {
      options: {
        responsive: true,
        maintainAspectRatio: false,
        animation: {
          duration: 0,
        },
        scales: {
          xAxes: [{
            type: 'time',
            time: {
              stepSize: 60,
            }
          }],
          yAxes: [{
            ticks: {
              min: 0,
              beginAtZero: true
            }
          }]
        },
        legend: {
          display: false
        },
        layout: {
          padding: 20
        }
      }
    }
  }
})
</script>


<style>
canvas {
  max-height: 300px !important;
}
</style>
