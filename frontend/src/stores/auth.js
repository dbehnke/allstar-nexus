import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('nexus_token') || '')
  const userRole = ref('')
  const authed = ref(!!token.value)

  const isAuthenticated = computed(() => authed.value && !!token.value)
  const isAdmin = computed(() => userRole.value === 'admin' || userRole.value === 'superadmin')

  function setToken(newToken, role) {
    token.value = newToken
    userRole.value = role
    authed.value = true
    localStorage.setItem('nexus_token', newToken)
  }

  function clearAuth() {
    token.value = ''
    userRole.value = ''
    authed.value = false
    localStorage.removeItem('nexus_token')
  }

  function getAuthHeaders() {
    if (!token.value) return {}
    return { 'Authorization': `Bearer ${token.value}` }
  }

  async function login(email, password) {
    try {
      const response = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
      })

      const data = await response.json()

      if (!response.ok || !data.ok) {
        throw new Error((data.error && data.error.message) || 'Login failed')
      }

      setToken(data.data.token, data.data.role)
      return true
    } catch (e) {
      throw e
    }
  }

  return {
    token,
    userRole,
    authed,
    isAuthenticated,
    isAdmin,
    setToken,
    clearAuth,
    getAuthHeaders,
    login
  }
})
