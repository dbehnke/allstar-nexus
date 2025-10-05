<template>
  <div class="app">
    <nav class="navbar">
      <div class="navbar-brand">
        <router-link to="/" class="brand-link">
          <h1>Allstar Nexus</h1>
          <span class="version" v-if="status && status.version">v{{ status.version }}</span>
        </router-link>
      </div>

      <div class="navbar-menu">
        <router-link to="/" class="nav-link">Dashboard</router-link>
        <router-link to="/lookup" class="nav-link">Node Lookup</router-link>
        <router-link to="/rpt-stats" class="nav-link" v-if="authStore.isAuthenticated">RPT Stats</router-link>
        <router-link to="/voter" class="nav-link" v-if="authStore.isAuthenticated">Voter</router-link>
      </div>

      <div class="navbar-user">
        <div v-if="!authStore.authed" class="login-toggle">
          <button @click="showLogin = !showLogin" class="btn-secondary">
            {{ showLogin ? 'Hide Login' : 'Admin Login' }}
          </button>
        </div>
        <div v-else class="user-info">
          <span class="user-role">{{ authStore.userRole }}</span>
          <button @click="logout" class="btn-secondary">Logout</button>
        </div>
      </div>
    </nav>

    <div v-if="showLogin && !authStore.authed" class="login-panel">
      <div class="login-container">
        <h3>Admin Login</h3>
        <form @submit.prevent="login">
          <div class="field">
            <label>Email</label>
            <input v-model="email" type="email" required />
          </div>
          <div class="field">
            <label>Password</label>
            <input v-model="password" type="password" required />
          </div>
          <div class="actions">
            <button type="submit" :disabled="loggingIn" class="btn-primary">
              {{ loggingIn ? 'Logging in...' : 'Login' }}
            </button>
          </div>
          <div v-if="loginError" class="error">{{ loginError }}</div>
        </form>
      </div>
    </div>

    <main class="main-content">
      <router-view />
    </main>

    <footer class="footer">
      <div class="footer-content">
        <p>&copy; 2025 Allstar Nexus. Built with modern Vue.js</p>
        <p v-if="status && status.build_time">
          Build: {{ new Date(status.build_time).toLocaleString() }}
        </p>
      </div>
    </footer>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useNodeStore } from './stores/node'

const router = useRouter()
const authStore = useAuthStore()
const nodeStore = useNodeStore()

const showLogin = ref(false)
const email = ref('')
const password = ref('')
const loggingIn = ref(false)
const loginError = ref('')

const status = computed(() => nodeStore.status)

async function login() {
  loggingIn.value = true
  loginError.value = ''

  try {
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: email.value, password: password.value })
    })

    const data = await response.json()

    if (!response.ok || !data.ok) {
      loginError.value = (data.error && data.error.message) || 'Login failed'
      return
    }

    authStore.setToken(data.data.token, data.data.role)
    showLogin.value = false
    email.value = ''
    password.value = ''
  } catch (e) {
    loginError.value = 'Network error'
    console.error('Login error:', e)
  } finally {
    loggingIn.value = false
  }
}

function logout() {
  authStore.clearAuth()
  router.push('/')
}

// Check for existing token on mount
if (authStore.token) {
  authStore.authed = true
}
</script>

<style>
* {
  box-sizing: border-box;
}

body {
  font-family: system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  margin: 0;
  background: #0f0f0f;
  color: #eee;
  line-height: 1.6;
}

.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

/* Navbar */
.navbar {
  background: #1c1c1c;
  border-bottom: 1px solid #333;
  padding: 1rem 2rem;
  display: flex;
  align-items: center;
  gap: 2rem;
  position: sticky;
  top: 0;
  z-index: 100;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

.navbar-brand h1 {
  margin: 0;
  font-size: 1.5rem;
  color: #60a5fa;
}

.brand-link {
  text-decoration: none;
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
}

.version {
  font-size: 0.875rem;
  color: #999;
  font-weight: 400;
}

.navbar-menu {
  flex: 1;
  display: flex;
  gap: 1.5rem;
}

.nav-link {
  color: #ddd;
  text-decoration: none;
  padding: 0.5rem 1rem;
  border-radius: 6px;
  transition: all 0.2s;
  font-weight: 500;
}

.nav-link:hover {
  background: #2a2a2a;
  color: #fff;
}

.nav-link.router-link-active {
  background: #3b82f6;
  color: #fff;
}

.navbar-user {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.user-role {
  color: #4ade80;
  font-weight: 600;
  text-transform: uppercase;
  font-size: 0.875rem;
}

.btn-primary {
  background: linear-gradient(135deg, #3b82f6, #1d4ed8);
  color: #fff;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 1rem;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: #374151;
  color: #eee;
  border: 1px solid #555;
  padding: 0.5rem 1rem;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-secondary:hover {
  background: #4b5563;
  border-color: #666;
}

/* Login Panel */
.login-panel {
  background: linear-gradient(135deg, #1e293b, #0f172a);
  border-bottom: 1px solid #333;
  padding: 2rem;
}

.login-container {
  max-width: 400px;
  margin: 0 auto;
  background: #1c1c1c;
  padding: 2rem;
  border: 1px solid #333;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
}

.login-container h3 {
  margin-top: 0;
  margin-bottom: 1.5rem;
  color: #60a5fa;
}

.field {
  margin-bottom: 1rem;
}

.field label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 600;
  color: #ddd;
}

.field input {
  width: 100%;
  background: #222;
  color: #eee;
  border: 1px solid #444;
  padding: 0.75rem;
  border-radius: 6px;
  font-size: 1rem;
  transition: border-color 0.2s;
}

.field input:focus {
  outline: none;
  border-color: #60a5fa;
}

.actions {
  margin-top: 1.5rem;
}

.actions button {
  width: 100%;
}

.error {
  color: #f87171;
  margin-top: 1rem;
  padding: 0.75rem;
  background: #7f1d1d;
  border-radius: 6px;
  font-size: 0.875rem;
}

/* Main Content */
.main-content {
  flex: 1;
}

/* Footer */
.footer {
  background: #1c1c1c;
  border-top: 1px solid #333;
  padding: 1.5rem 2rem;
  margin-top: 3rem;
}

.footer-content {
  max-width: 1400px;
  margin: 0 auto;
  text-align: center;
  color: #999;
  font-size: 0.875rem;
}

.footer-content p {
  margin: 0.25rem 0;
}

/* Responsive */
@media (max-width: 768px) {
  .navbar {
    flex-direction: column;
    gap: 1rem;
  }

  .navbar-menu {
    flex-wrap: wrap;
    justify-content: center;
  }

  .navbar-user {
    width: 100%;
    justify-content: center;
  }
}
</style>
