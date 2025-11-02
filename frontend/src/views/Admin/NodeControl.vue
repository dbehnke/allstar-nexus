<template>
  <div class="node-control">
    <div class="panel-header">
      <h2>Node Control</h2>
      <p class="subtitle">Link and unlink nodes via AMI</p>
    </div>

    <div class="control-section">
      <div class="link-panel">
        <h3>Link Node</h3>
        <div class="form-group">
          <label for="link-local-node">Local Node:</label>
          <input
            id="link-local-node"
            v-model="linkForm.localNode"
            type="number"
            placeholder="e.g., 12345"
          />
        </div>
        <div class="form-group">
          <label for="link-remote-node">Remote Node:</label>
          <input
            id="link-remote-node"
            v-model="linkForm.remoteNode"
            type="number"
            placeholder="e.g., 54321"
          />
        </div>
        <div class="form-group">
          <label>
            <input type="checkbox" v-model="linkForm.permanent" />
            Permanent Link (adds to node stanza)
          </label>
        </div>
        <button
          @click="linkNode"
          :disabled="!canLink || linking"
          class="btn-primary"
        >
          <span v-if="linking">Linking...</span>
          <span v-else>Link Node</span>
        </button>
      </div>

      <div class="unlink-panel">
        <h3>Unlink Node</h3>
        <div class="form-group">
          <label for="unlink-local-node">Local Node:</label>
          <input
            id="unlink-local-node"
            v-model="unlinkForm.localNode"
            type="number"
            placeholder="e.g., 12345"
          />
        </div>
        <div class="form-group">
          <label for="unlink-remote-node">Remote Node:</label>
          <input
            id="unlink-remote-node"
            v-model="unlinkForm.remoteNode"
            type="number"
            placeholder="e.g., 54321"
          />
        </div>
        <button
          @click="unlinkNode"
          :disabled="!canUnlink || unlinking"
          class="btn-danger"
        >
          <span v-if="unlinking">Unlinking...</span>
          <span v-else>Unlink Node</span>
        </button>
      </div>
    </div>

    <div v-if="message" :class="['message-box', message.type]">
      {{ message.text }}
    </div>

    <div v-if="nodeHistory.length > 0" class="history-section">
      <h3>Recent Actions</h3>
      <div class="history-list">
        <div
          v-for="(item, index) in nodeHistory"
          :key="index"
          :class="['history-item', item.success ? 'success' : 'error']"
        >
          <div class="history-header">
            <span class="timestamp">{{ formatTimestamp(item.timestamp) }}</span>
            <span class="action-type">{{ item.action }}</span>
            <span class="node-info">
              {{ item.localNode }} {{ item.action === 'Link' ? '→' : '↚' }} {{ item.remoteNode }}
            </span>
            <span :class="['status-badge', item.success ? 'success' : 'error']">
              {{ item.success ? 'Success' : 'Failed' }}
            </span>
          </div>
          <div v-if="item.message" class="history-message">
            <small>{{ item.message }}</small>
          </div>
        </div>
      </div>
    </div>

    <div class="info-section">
      <h3>ℹ️ Information</h3>
      <ul>
        <li><strong>Link Node:</strong> Creates a temporary or permanent connection between nodes</li>
        <li><strong>Unlink Node:</strong> Disconnects a linked node</li>
        <li><strong>Permanent:</strong> When checked, the link is added to the node's configuration (persists across restarts)</li>
        <li><strong>Temporary:</strong> When unchecked, the link is active only until disconnected or node restart</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { apiRequest } from '../../utils/api'

interface LinkForm {
  localNode: string
  remoteNode: string
  permanent: boolean
}

interface UnlinkForm {
  localNode: string
  remoteNode: string
}

interface Message {
  type: 'success' | 'error'
  text: string
}

interface HistoryItem {
  timestamp: Date
  action: 'Link' | 'Unlink'
  localNode: string
  remoteNode: string
  success: boolean
  message?: string
}

const linkForm = ref<LinkForm>({
  localNode: '',
  remoteNode: '',
  permanent: false
})

const unlinkForm = ref<UnlinkForm>({
  localNode: '',
  remoteNode: ''
})

const linking = ref(false)
const unlinking = ref(false)
const message = ref<Message | null>(null)
const nodeHistory = ref<HistoryItem[]>([])

const canLink = computed(() => {
  return linkForm.value.localNode.trim() !== '' && linkForm.value.remoteNode.trim() !== ''
})

const canUnlink = computed(() => {
  return unlinkForm.value.localNode.trim() !== '' && unlinkForm.value.remoteNode.trim() !== ''
})

const linkNode = async () => {
  if (!canLink.value) return

  linking.value = true
  message.value = null

  try {
    const response = await apiRequest<{ success: boolean; output: string }>(
      '/api/admin/ami/link',
      {
        method: 'POST',
        body: JSON.stringify({
          local_node: linkForm.value.localNode,
          remote_node: linkForm.value.remoteNode,
          permanent: linkForm.value.permanent
        })
      }
    )

    message.value = {
      type: response.success ? 'success' : 'error',
      text: response.output
    }

    // Add to history
    nodeHistory.value.unshift({
      timestamp: new Date(),
      action: 'Link',
      localNode: linkForm.value.localNode,
      remoteNode: linkForm.value.remoteNode,
      success: response.success,
      message: response.output
    })

    // Keep only last 20 items
    if (nodeHistory.value.length > 20) {
      nodeHistory.value = nodeHistory.value.slice(0, 20)
    }

    // Clear form on success
    if (response.success) {
      linkForm.value = {
        localNode: '',
        remoteNode: '',
        permanent: false
      }
    }
  } catch (error: any) {
    message.value = {
      type: 'error',
      text: error.message || 'Failed to link node'
    }
  } finally {
    linking.value = false
  }
}

const unlinkNode = async () => {
  if (!canUnlink.value) return

  unlinking.value = true
  message.value = null

  try {
    const response = await apiRequest<{ success: boolean; output: string }>(
      '/api/admin/ami/unlink',
      {
        method: 'POST',
        body: JSON.stringify({
          local_node: unlinkForm.value.localNode,
          remote_node: unlinkForm.value.remoteNode
        })
      }
    )

    message.value = {
      type: response.success ? 'success' : 'error',
      text: response.output
    }

    // Add to history
    nodeHistory.value.unshift({
      timestamp: new Date(),
      action: 'Unlink',
      localNode: unlinkForm.value.localNode,
      remoteNode: unlinkForm.value.remoteNode,
      success: response.success,
      message: response.output
    })

    // Keep only last 20 items
    if (nodeHistory.value.length > 20) {
      nodeHistory.value = nodeHistory.value.slice(0, 20)
    }

    // Clear form on success
    if (response.success) {
      unlinkForm.value = {
        localNode: '',
        remoteNode: ''
      }
    }
  } catch (error: any) {
    message.value = {
      type: 'error',
      text: error.message || 'Failed to unlink node'
    }
  } finally {
    unlinking.value = false
  }
}

const formatTimestamp = (date: Date) => {
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(date)
}
</script>

<style scoped>
.node-control {
  max-width: 1000px;
  margin: 0 auto;
  padding: 20px;
}

.panel-header {
  margin-bottom: 30px;
}

.panel-header h2 {
  margin: 0 0 10px 0;
  font-size: 28px;
  color: #2c3e50;
}

.subtitle {
  margin: 0;
  color: #7f8c8d;
  font-size: 14px;
}

.control-section {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 25px;
  margin-bottom: 25px;
}

@media (max-width: 768px) {
  .control-section {
    grid-template-columns: 1fr;
  }
}

.link-panel,
.unlink-panel {
  background: #ffffff;
  border-radius: 8px;
  padding: 25px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.link-panel h3,
.unlink-panel h3 {
  margin: 0 0 20px 0;
  font-size: 18px;
  color: #2c3e50;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-weight: 600;
  color: #34495e;
  font-size: 14px;
}

.form-group input[type="number"],
.form-group input[type="text"] {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #dce1e6;
  border-radius: 6px;
  font-size: 14px;
  transition: border-color 0.2s;
}

.form-group input[type="number"]:focus,
.form-group input[type="text"]:focus {
  outline: none;
  border-color: #3498db;
}

.form-group input[type="checkbox"] {
  margin-right: 8px;
}

.btn-primary,
.btn-danger {
  width: 100%;
  padding: 12px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary {
  background: #3498db;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #2980b9;
}

.btn-primary:disabled {
  background: #95a5a6;
  cursor: not-allowed;
}

.btn-danger {
  background: #e74c3c;
  color: white;
}

.btn-danger:hover:not(:disabled) {
  background: #c0392b;
}

.btn-danger:disabled {
  background: #95a5a6;
  cursor: not-allowed;
}

.message-box {
  padding: 15px;
  border-radius: 6px;
  margin-bottom: 25px;
  font-size: 14px;
}

.message-box.success {
  background: #d5f4e6;
  border: 1px solid #27ae60;
  color: #145a32;
}

.message-box.error {
  background: #fadbd8;
  border: 1px solid #e74c3c;
  color: #7b241c;
}

.history-section {
  background: #ffffff;
  border-radius: 8px;
  padding: 25px;
  margin-bottom: 25px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.history-section h3 {
  margin: 0 0 15px 0;
  font-size: 16px;
  color: #2c3e50;
}

.history-list {
  max-height: 400px;
  overflow-y: auto;
}

.history-item {
  padding: 12px;
  margin-bottom: 10px;
  border-radius: 6px;
  border-left: 4px solid;
}

.history-item.success {
  background: #f0f9f4;
  border-left-color: #27ae60;
}

.history-item.error {
  background: #fef5f5;
  border-left-color: #e74c3c;
}

.history-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 5px;
  flex-wrap: wrap;
}

.timestamp {
  font-size: 12px;
  color: #7f8c8d;
}

.action-type {
  font-weight: 600;
  font-size: 13px;
  color: #2c3e50;
}

.node-info {
  font-size: 13px;
  color: #34495e;
  font-family: 'Courier New', monospace;
}

.status-badge {
  padding: 3px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 600;
}

.status-badge.success {
  background: #d5f4e6;
  color: #27ae60;
}

.status-badge.error {
  background: #fadbd8;
  color: #e74c3c;
}

.history-message {
  font-size: 12px;
  color: #7f8c8d;
}

.info-section {
  background: #e8f4f8;
  border: 1px solid #3498db;
  border-radius: 8px;
  padding: 20px;
}

.info-section h3 {
  margin: 0 0 15px 0;
  font-size: 16px;
  color: #2c3e50;
}

.info-section ul {
  margin: 0;
  padding-left: 20px;
}

.info-section li {
  margin-bottom: 10px;
  color: #34495e;
  font-size: 13px;
  line-height: 1.6;
}

.info-section li strong {
  color: #2c3e50;
}
</style>
