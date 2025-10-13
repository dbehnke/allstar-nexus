import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'

// In E2E TEST_MODE, load dev test helpers early so tests can inject WS envelopes
try {
	if (typeof window !== 'undefined' && window.__NEXUS_CONFIG__ && window.__NEXUS_CONFIG__.TEST_MODE) {
		import('./dev/test-helpers.js').then(m => { try { m && m.default && m.default() } catch (e) {} }).catch(() => {})
	}
} catch (e) {}

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.mount('#app')