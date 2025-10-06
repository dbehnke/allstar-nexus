import { ref, watch, onMounted } from 'vue'

const STORAGE_KEY = 'allstar-nexus-theme'
const theme = ref('system') // 'light', 'dark', or 'system'

export function useTheme() {
  const isDark = ref(false)

  function getSystemPreference() {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  }

  function applyTheme() {
    let shouldBeDark = false

    if (theme.value === 'system') {
      shouldBeDark = getSystemPreference()
    } else {
      shouldBeDark = theme.value === 'dark'
    }

    isDark.value = shouldBeDark

    if (shouldBeDark) {
      document.documentElement.classList.add('dark')
      document.documentElement.classList.remove('light')
    } else {
      document.documentElement.classList.add('light')
      document.documentElement.classList.remove('dark')
    }
  }

  function setTheme(newTheme) {
    theme.value = newTheme
    localStorage.setItem(STORAGE_KEY, newTheme)
    applyTheme()
  }

  function initTheme() {
    // Load from localStorage or default to system
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored && ['light', 'dark', 'system'].includes(stored)) {
      theme.value = stored
    }

    applyTheme()

    // Listen for system theme changes
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    mediaQuery.addEventListener('change', () => {
      if (theme.value === 'system') {
        applyTheme()
      }
    })
  }

  onMounted(() => {
    initTheme()
  })

  watch(theme, applyTheme)

  return {
    theme,
    isDark,
    setTheme,
    getSystemPreference
  }
}
