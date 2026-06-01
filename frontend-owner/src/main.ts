import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from '@/app/App.vue'
import { router } from '@/router'
import '@/app/styles.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)

// Register global error handler for telemetry
app.config.errorHandler = (err, _instance, info) => {
  console.error('[vue] unhandled error:', err, 'info:', info)
  // TODO: send to Sentry or `/api/v1/telemetry/errors` in production
}

app.mount('#app')
