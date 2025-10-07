<template>
  <div class="app">
    <nav class="navbar">
      <div class="navbar-header">
        <div class="navbar-brand">
          <router-link to="/" class="brand-link">
            <div class="brand-text">
              <h1>{{ status?.title || 'Allstar Nexus' }}</h1>
              <span class="subtitle" v-if="status?.subtitle">{{ status.subtitle }}</span>
            </div>
          </router-link>
        </div>

        <button @click="mobileMenuOpen = !mobileMenuOpen" class="mobile-menu-toggle" aria-label="Toggle menu">
          <span class="hamburger">☰</span>
        </button>
      </div>

      <div class="navbar-menu" :class="{ 'mobile-open': mobileMenuOpen }">
        <router-link to="/" class="nav-link" @click="mobileMenuOpen = false">Dashboard</router-link>
        <router-link to="/status" class="nav-link" @click="mobileMenuOpen = false">Node Status</router-link>
        <router-link to="/lookup" class="nav-link" @click="mobileMenuOpen = false">Node Lookup</router-link>
        <router-link to="/rpt-stats" class="nav-link" v-if="authStore.isAuthenticated" @click="mobileMenuOpen = false">RPT Stats</router-link>
        <router-link to="/voter" class="nav-link" v-if="authStore.isAuthenticated" @click="mobileMenuOpen = false">Voter</router-link>
      </div>

      <div class="navbar-user" :class="{ 'mobile-open': mobileMenuOpen }">
        <button @click="cycleTheme" class="btn-icon theme-toggle" :title="themeTooltip">
          <span v-if="theme === 'light'">☀️</span>
          <span v-else-if="theme === 'dark'">🌙</span>
          <span v-else>🖥️</span>
        </button>

        <div v-if="!authStore.authed" class="login-toggle">
          <button @click="showLogin = !showLogin; mobileMenuOpen = false" class="btn-secondary">
            {{ showLogin ? 'Hide Login' : 'Admin Login' }}
          </button>
        </div>
        <div v-else class="user-info">
          <span class="user-role">{{ authStore.userRole }}</span>
          <button @click="logout; mobileMenuOpen = false" class="btn-secondary">Logout</button>
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
  <p>&copy; 2025 Allstar Nexus<span v-if="status && status.version">&nbsp;{{ status.version }}</span>. Built with ❤️ in Macomb, MI</p>
        <p v-if="status && status.build_time">
          Build: {{ new Date(status.build_time).toLocaleString() }}
        </p>
      </div>
    </footer>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useNodeStore } from './stores/node'
import { useTheme } from './composables/useTheme'

const router = useRouter()
const authStore = useAuthStore()
const nodeStore = useNodeStore()
const { theme, setTheme } = useTheme()

const mobileMenuOpen = ref(false)
const showLogin = ref(false)
const email = ref('')
const password = ref('')
const loggingIn = ref(false)
const loginError = ref('')

const status = computed(() => nodeStore.status)

const themeTooltip = computed(() => {
  if (theme.value === 'light') return 'Switch to dark mode'
  if (theme.value === 'dark') return 'Switch to system theme'
  return 'Switch to light mode'
})

function cycleTheme() {
  if (theme.value === 'light') {
    setTheme('dark')
  } else if (theme.value === 'dark') {
    setTheme('system')
  } else {
    setTheme('light')
  }
}

// Update document title when status changes
watch(status, (newStatus) => {
  if (newStatus?.title) {
    if (newStatus.subtitle) {
      document.title = `${newStatus.title} - ${newStatus.subtitle}`
    } else {
      document.title = newStatus.title
    }
  }
}, { immediate: true })

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

:root {
  /* Dark theme (default) */
  --bg-primary: #0f0f0f;
  --bg-secondary: #1c1c1c;
  --bg-tertiary: #222;
  --bg-hover: #2a2a2a;
  --bg-input: #222;

  --text-primary: #eee;
  --text-secondary: #ddd;
  --text-muted: #999;
  --text-label: #999;

  --border-color: #333;
  --border-hover: #555;

  --accent-primary: #60a5fa;
  --accent-hover: #3b82f6;
  --accent-gradient-start: #3b82f6;
  --accent-gradient-end: #1d4ed8;

  --success: #4ade80;
  --error: #f87171;
  --warning: #fbbf24;

  --shadow: rgba(0, 0, 0, 0.3);
  --shadow-strong: rgba(0, 0, 0, 0.4);
}

:root.light {
  /* Light theme - Natural, easy on the eyes */
  --bg-primary: #f5f7fa;
  --bg-secondary: #ffffff;
  --bg-tertiary: #edf2f7;
  --bg-hover: #e2e8f0;
  --bg-input: #ffffff;

  --text-primary: #2d3748;
  --text-secondary: #4a5568;
  --text-muted: #718096;
  --text-label: #4a5568;

  --border-color: #cbd5e0;
  --border-hover: #a0aec0;

  --accent-primary: #2b6cb0;
  --accent-hover: #2c5282;
  --accent-gradient-start: #3182ce;
  --accent-gradient-end: #2c5282;

  --success: #38a169;
  --error: #e53e3e;
  --warning: #dd6b20;

  --shadow: rgba(0, 0, 0, 0.05);
  --shadow-strong: rgba(0, 0, 0, 0.1);
}

body {
  font-family: system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  margin: 0;
  background: var(--bg-primary);
  color: var(--text-primary);
  line-height: 1.6;
  transition: background-color 0.3s ease, color 0.3s ease;
}

.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

/* Navbar */
.navbar {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  padding: 1rem 2rem;
  display: flex;
  align-items: center;
  gap: 2rem;
  position: sticky;
  top: 0;
  z-index: 100;
  box-shadow: 0 2px 8px var(--shadow);
  transition: background-color 0.3s ease, border-color 0.3s ease;
}

.navbar-brand h1 {
  margin: 0;
  font-size: 1.5rem;
  color: var(--accent-primary);
}

.brand-link {
  text-decoration: none;
  display: block;
}

.brand-text {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.subtitle {
  font-size: 0.875rem;
  color: var(--text-muted);
  font-weight: 400;
  line-height: 1.2;
}

.navbar-menu {
  flex: 1;
  display: flex;
  gap: 1.5rem;
}

.nav-link {
  color: var(--text-secondary);
  text-decoration: none;
  padding: 0.5rem 1rem;
  border-radius: 6px;
  transition: all 0.2s;
  font-weight: 500;
}

.nav-link:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.nav-link.router-link-active {
  background: var(--accent-hover);
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
  color: var(--success);
  font-weight: 600;
  text-transform: uppercase;
  font-size: 0.875rem;
}

.btn-primary {
  background: linear-gradient(135deg, var(--accent-gradient-start), var(--accent-gradient-end));
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
  box-shadow: 0 4px 12px var(--shadow);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border: 1px solid var(--border-hover);
  padding: 0.5rem 1rem;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-secondary:hover {
  background: var(--bg-hover);
  border-color: var(--border-hover);
}

.btn-icon {
  background: transparent;
  border: 1px solid var(--border-hover);
  color: var(--text-primary);
  width: 36px;
  height: 36px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 1.2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-icon:hover {
  background: var(--bg-hover);
  border-color: var(--border-hover);
}

.theme-toggle {
  font-size: 1.3rem;
}

/* Login Panel */
.login-panel {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  padding: 2rem;
}

.login-container {
  max-width: 400px;
  margin: 0 auto;
  background: var(--bg-secondary);
  padding: 2rem;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 16px var(--shadow-strong);
}

.login-container h3 {
  margin-top: 0;
  margin-bottom: 1.5rem;
  color: var(--accent-primary);
}

.field {
  margin-bottom: 1rem;
}

.field label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.field input {
  width: 100%;
  background: var(--bg-input);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  padding: 0.75rem;
  border-radius: 6px;
  font-size: 1rem;
  transition: border-color 0.2s;
}

.field input:focus {
  outline: none;
  border-color: var(--accent-primary);
}

.actions {
  margin-top: 1.5rem;
}

.actions button {
  width: 100%;
}

.error {
  color: var(--error);
  margin-top: 1rem;
  padding: 0.75rem;
  background: color-mix(in srgb, var(--error) 20%, var(--bg-secondary));
  border-radius: 6px;
  font-size: 0.875rem;
}

/* Main Content */
.main-content {
  flex: 1;
}

/* Footer */
.footer {
  background: var(--bg-secondary);
  border-top: 1px solid var(--border-color);
  padding: 1.5rem 2rem;
  margin-top: 3rem;
}

.footer-content {
  max-width: 1400px;
  margin: 0 auto;
  text-align: center;
  color: var(--text-muted);
  font-size: 0.875rem;
}

.footer-content p {
  margin: 0.25rem 0;
}

/* Mobile Menu */
.navbar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.mobile-menu-toggle {
  display: none;
  background: transparent;
  border: 1px solid var(--border-hover);
  color: var(--text-primary);
  width: 40px;
  height: 40px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 1.5rem;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.mobile-menu-toggle:hover {
  background: var(--bg-hover);
}

.hamburger {
  display: block;
  line-height: 1;
}

/* Responsive */
@media (max-width: 768px) {
  .navbar {
    flex-direction: column;
    gap: 0;
    padding: 1rem;
  }

  .navbar-header {
    width: 100%;
  }

  .mobile-menu-toggle {
    display: flex;
  }

  .navbar-menu {
    display: none;
    width: 100%;
    flex-direction: column;
    gap: 0.5rem;
    padding: 1rem 0 0.5rem 0;
  }

  .navbar-menu.mobile-open {
    display: flex;
  }

  .navbar-user {
    display: none;
    width: 100%;
    justify-content: center;
    padding: 0.5rem 0;
    border-top: 1px solid var(--border-color);
    margin-top: 0.5rem;
  }

  .navbar-user.mobile-open {
    display: flex;
  }

  .nav-link {
    padding: 0.75rem 1rem;
    text-align: center;
  }
}

@media (min-width: 769px) {
  .navbar {
    flex-direction: row;
  }

  .navbar-header {
    width: auto;
  }

  .navbar-menu {
    display: flex;
  }

  .navbar-user {
    display: flex;
  }
}
</style>
