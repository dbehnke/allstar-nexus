<template>
  <div class="config-editor">
    <div class="editor-header">
      <h1>Configuration Editor</h1>
      <p class="subtitle">Edit config.yaml with validation and automatic backups</p>
    </div>

    <div class="editor-toolbar">
      <button @click="loadConfig" class="btn-secondary" :disabled="loading">
        <span v-if="!loading">🔄 Reload</span>
        <span v-else>Loading...</span>
      </button>
      <button @click="showBackups = !showBackups" class="btn-secondary">
        📦 Backups ({{ backups.length }})
      </button>
      <button @click="validateConfig" class="btn-secondary" :disabled="!hasChanges">
        ✓ Validate
      </button>
      <button @click="saveConfig" class="btn-primary" :disabled="!hasChanges || saving">
        <span v-if="!saving">💾 Save Config</span>
        <span v-else">Saving...</span>
      </button>
    </div>

    <!-- Validation Status -->
    <div v-if="validationError" class="validation-error">
      <strong>❌ Validation Error:</strong> {{ validationError }}
    </div>
    <div v-if="validationSuccess" class="validation-success">
      ✓ Configuration is valid!
    </div>

    <!-- Main Editor Area -->
    <div class="editor-container">
      <!-- YAML Editor -->
      <div class="editor-pane">
        <div class="pane-header">
          <h3>config.yaml</h3>
          <span v-if="hasChanges" class="unsaved-indicator">● Unsaved changes</span>
        </div>
        <textarea
          v-model="configContent"
          class="yaml-editor"
          spellcheck="false"
          @input="onContentChange"
        ></textarea>
        <div class="editor-footer">
          <span class="line-count">{{ lineCount }} lines</span>
          <span class="char-count">{{ configContent.length }} characters</span>
        </div>
      </div>

      <!-- Backups Sidebar -->
      <div v-if="showBackups" class="backups-pane">
        <div class="pane-header">
          <h3>Config Backups</h3>
          <button @click="loadBackups" class="btn-icon">🔄</button>
        </div>
        <div class="backups-list">
          <div v-if="backups.length === 0" class="no-backups">
            No backups available
          </div>
          <div
            v-for="backup in backups"
            :key="backup.id"
            class="backup-item"
          >
            <div class="backup-info">
              <div class="backup-time">{{ formatDate(backup.timestamp) }}</div>
              <div class="backup-id">{{ backup.id }}</div>
              <div v-if="backup.comment" class="backup-comment">{{ backup.comment }}</div>
              <div class="backup-size">{{ formatSize(backup.size) }}</div>
            </div>
            <div class="backup-actions">
              <button @click="restoreBackup(backup.id)" class="btn-small">Restore</button>
              <button @click="viewBackup(backup.id)" class="btn-small">View</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Save Comment Modal -->
    <div v-if="showCommentModal" class="modal-overlay" @click="showCommentModal = false">
      <div class="modal" @click.stop>
        <h3>Save Configuration</h3>
        <p>Add an optional comment for this change:</p>
        <textarea
          v-model="saveComment"
          placeholder="e.g., Updated node configuration"
          rows="3"
          class="comment-input"
        ></textarea>
        <div class="modal-actions">
          <button @click="showCommentModal = false" class="btn-secondary">Cancel</button>
          <button @click="confirmSave" class="btn-primary">Save</button>
        </div>
      </div>
    </div>

    <!-- Toast Notifications -->
    <div v-if="toast" class="toast" :class="toast.type">
      {{ toast.message }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { logger } from '../../utils/logger'

const configContent = ref('')
const originalContent = ref('')
const backups = ref([])
const showBackups = ref(false)
const loading = ref(false)
const saving = ref(false)
const validationError = ref('')
const validationSuccess = ref(false)
const showCommentModal = ref(false)
const saveComment = ref('')
const toast = ref(null)

const hasChanges = computed(() => configContent.value !== originalContent.value)
const lineCount = computed(() => configContent.value.split('\n').length)

async function loadConfig() {
  loading.value = true
  try {
    const response = await fetch('/api/admin/config', {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    const data = await response.json()
    if (data.ok !== false && data.data) {
      configContent.value = data.data.content
      originalContent.value = data.data.content
      showToast('Config loaded successfully', 'success')
    } else {
      showToast('Failed to load config', 'error')
    }
  } catch (e) {
    logger.error('Failed to load config', e)
    showToast('Network error loading config', 'error')
  } finally {
    loading.value = false
  }
}

async function loadBackups() {
  try {
    const response = await fetch('/api/admin/config/backups', {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    const data = await response.json()
    if (data.ok !== false && data.data) {
      backups.value = data.data
    }
  } catch (e) {
    logger.error('Failed to load backups', e)
  }
}

function onContentChange() {
  validationError.value = ''
  validationSuccess.value = false
}

async function validateConfig() {
  // Simple client-side validation (check for tabs)
  const lines = configContent.value.split('\n')
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].match(/^\s*\t/)) {
      validationError.value = `Line ${i + 1}: YAML does not allow tabs for indentation. Use spaces.`
      return
    }
  }
  validationSuccess.value = true
  setTimeout(() => {
    validationSuccess.value = false
  }, 3000)
}

function saveConfig() {
  showCommentModal.value = true
}

async function confirmSave() {
  showCommentModal.value = false
  saving.value = true

  try {
    const response = await fetch('/api/admin/config', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        content: configContent.value,
        comment: saveComment.value
      })
    })

    const data = await response.json()

    if (data.ok !== false && data.data && data.data.success) {
      originalContent.value = configContent.value
      saveComment.value = ''
      showToast(`Config saved! Backup ID: ${data.data.backup_id}`, 'success')
      loadBackups()
    } else {
      const errorMsg = data.error?.message || 'Save failed'
      validationError.value = errorMsg
      showToast(errorMsg, 'error')
    }
  } catch (e) {
    logger.error('Failed to save config', e)
    showToast('Network error saving config', 'error')
  } finally {
    saving.value = false
  }
}

async function restoreBackup(backupId) {
  if (!confirm(`Restore config from backup ${backupId}? This will create a backup of the current config first.`)) {
    return
  }

  try {
    const response = await fetch(`/api/admin/config/restore/${backupId}`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })

    const data = await response.json()

    if (data.ok !== false && data.data && data.data.success) {
      showToast('Config restored successfully', 'success')
      loadConfig()
      loadBackups()
    } else {
      showToast(data.error?.message || 'Restore failed', 'error')
    }
  } catch (e) {
    logger.error('Failed to restore backup', e)
    showToast('Network error restoring backup', 'error')
  }
}

async function viewBackup(backupId) {
  // For now, just show in editor (could open in modal later)
  try {
    const response = await fetch(`/api/admin/config/diff?from=${backupId}&to=current`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    const data = await response.json()
    if (data.ok !== false && data.data) {
      // Show diff in console for now
      console.log('Diff from backup to current:', data.data.diff)
      showToast('Diff logged to console', 'info')
    }
  } catch (e) {
    logger.error('Failed to get diff', e)
  }
}

function formatDate(timestamp) {
  const date = new Date(timestamp)
  return date.toLocaleString()
}

function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function showToast(message, type = 'info') {
  toast.value = { message, type }
  setTimeout(() => {
    toast.value = null
  }, 4000)
}

onMounted(() => {
  loadConfig()
  loadBackups()
})
</script>

<style scoped>
.config-editor {
  padding: 2rem;
  max-width: 1600px;
  margin: 0 auto;
}

.editor-header {
  margin-bottom: 1.5rem;
}

.editor-header h1 {
  margin: 0 0 0.5rem 0;
  color: var(--text-primary);
}

.subtitle {
  margin: 0;
  color: var(--text-muted);
}

.editor-toolbar {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
  padding: 1rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
}

.validation-error {
  padding: 1rem;
  background: color-mix(in srgb, var(--error) 20%, var(--bg-secondary));
  border: 1px solid var(--error);
  border-radius: 8px;
  color: var(--error);
  margin-bottom: 1rem;
}

.validation-success {
  padding: 1rem;
  background: color-mix(in srgb, var(--success) 20%, var(--bg-secondary));
  border: 1px solid var(--success);
  border-radius: 8px;
  color: var(--success);
  margin-bottom: 1rem;
}

.editor-container {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 1rem;
  min-height: 600px;
}

.editor-pane, .backups-pane {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
}

.pane-header {
  padding: 1rem;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pane-header h3 {
  margin: 0;
  color: var(--text-primary);
}

.unsaved-indicator {
  color: var(--warning);
  font-weight: 600;
}

.yaml-editor {
  flex: 1;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 14px;
  line-height: 1.6;
  padding: 1rem;
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border: none;
  resize: none;
  outline: none;
  tab-size: 2;
}

.editor-footer {
  padding: 0.75rem 1rem;
  border-top: 1px solid var(--border-color);
  display: flex;
  gap: 2rem;
  font-size: 0.875rem;
  color: var(--text-muted);
}

.backups-pane {
  width: 350px;
}

.backups-list {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
}

.no-backups {
  text-align: center;
  color: var(--text-muted);
  padding: 2rem;
}

.backup-item {
  padding: 1rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  margin-bottom: 0.75rem;
  transition: border-color 0.2s;
}

.backup-item:hover {
  border-color: var(--accent-primary);
}

.backup-info {
  margin-bottom: 0.75rem;
}

.backup-time {
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.25rem;
}

.backup-id {
  font-size: 0.75rem;
  color: var(--text-muted);
  font-family: monospace;
}

.backup-comment {
  font-size: 0.875rem;
  color: var(--text-secondary);
  margin-top: 0.5rem;
  font-style: italic;
}

.backup-size {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
}

.backup-actions {
  display: flex;
  gap: 0.5rem;
}

.btn-small {
  padding: 0.25rem 0.75rem;
  font-size: 0.875rem;
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-small:hover {
  background: var(--accent-primary);
  color: #fff;
  border-color: var(--accent-primary);
}

.btn-icon {
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: 1.2rem;
  padding: 0.25rem;
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
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px var(--shadow);
}

.btn-secondary {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  padding: 0.75rem 1.5rem;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-secondary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary:hover:not(:disabled) {
  background: var(--bg-hover);
}

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 2rem;
  max-width: 500px;
  width: 90%;
}

.modal h3 {
  margin: 0 0 1rem 0;
  color: var(--text-primary);
}

.comment-input {
  width: 100%;
  padding: 0.75rem;
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-family: inherit;
  resize: vertical;
  margin: 1rem 0;
}

.modal-actions {
  display: flex;
  gap: 1rem;
  justify-content: flex-end;
}

/* Toast */
.toast {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  padding: 1rem 1.5rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 16px var(--shadow-strong);
  z-index: 1001;
  animation: slideIn 0.3s ease;
}

.toast.success {
  border-color: var(--success);
}

.toast.error {
  border-color: var(--error);
}

.toast.info {
  border-color: var(--accent-primary);
}

@keyframes slideIn {
  from {
    transform: translateX(400px);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

@media (max-width: 1024px) {
  .editor-container {
    grid-template-columns: 1fr;
  }

  .backups-pane {
    width: 100%;
    max-height: 400px;
  }
}
</style>
