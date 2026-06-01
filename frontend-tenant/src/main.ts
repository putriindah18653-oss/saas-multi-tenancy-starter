import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from '@/app/App.vue'
import { router } from '@/router'
import '@/app/styles.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)

// Global error handler for telemetry
app.config.errorHandler = (err, _instance, info) => {
  console.error('[vue] unhandled error:', err, 'info:', info)
}

// Capture unhandled promise rejections
window.addEventListener('unhandledrejection', (event) => {
  console.error('[vue] unhandled rejection:', event.reason)
})

app.mount('#app')
