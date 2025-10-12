import { defineStore } from 'pinia'
import { ref } from 'vue'

let _id = 1

export const useUIStore = defineStore('ui', () => {
  const toasts = ref([]) // [{ id, message, type, createdAt }]

  function addToast(message, opts = {}) {
    const id = _id++
    const type = opts.type || 'info'
    const duration = typeof opts.duration === 'number' ? opts.duration : 3500
    const toast = { id, message, type, createdAt: Date.now() }
    toasts.value.push(toast)
    if (duration > 0) {
      setTimeout(() => removeToast(id), duration)
    }
    return id
  }

  function removeToast(id) {
    const idx = toasts.value.findIndex(t => t.id === id)
    if (idx >= 0) toasts.value.splice(idx, 1)
  }

  return { toasts, addToast, removeToast }
})
