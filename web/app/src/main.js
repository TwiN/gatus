import { createApp } from 'vue'
import App from './App.vue'
import './index.css'

export const SERVER_URL = process.env.NODE_ENV === 'production' ? '.' : 'http://localhost:8080'

createApp(App).mount('#app')
