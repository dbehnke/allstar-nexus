<template>
  <div class="command-panel">
    <div class="panel-header">
      <h2>AMI Command Panel</h2>
      <p class="subtitle">Execute Asterisk Manager Interface commands</p>
    </div>

    <div class="command-section">
      <div class="form-group">
        <label for="command-select">Command:</label>
        <select id="command-select" v-model="selectedCommand" @change="onCommandChange">
          <option value="">-- Select a command --</option>
          <optgroup label="Predefined Commands">
            <option v-for="cmd in predefinedCommands" :key="cmd.name" :value="cmd.name">
              {{ cmd.label }}
            </option>
          </optgroup>
          <option value="custom">Custom Command</option>
        </select>
      </div>

      <div v-if="selectedCommand" class="command-details">
        <div v-if="selectedCommand === 'custom'" class="form-group">
          <label for="custom-action">Action:</label>
          <input
            id="custom-action"
            v-model="customAction"
            type="text"
            placeholder="e.g., Command"
          />
        </div>

        <div v-if="currentCommandParams.length > 0" class="params-section">
          <h3>Parameters</h3>
          <div v-for="param in currentCommandParams" :key="param.key" class="form-group">
            <label :for="`param-${param.key}`">{{ param.label }}:</label>
            <input
              :id="`param-${param.key}`"
              v-model="commandParams[param.key]"
              :type="param.type || 'text'"
              :placeholder="param.placeholder || ''"
            />
            <small v-if="param.description" class="param-description">
              {{ param.description }}
            </small>
          </div>
        </div>

        <div class="action-buttons">
          <button
            @click="executeCommand"
            :disabled="!canExecute || executing"
            class="btn-primary"
          >
            <span v-if="executing">Executing...</span>
            <span v-else>Execute Command</span>
          </button>
          <button @click="clearForm" class="btn-secondary">Clear</button>
        </div>
      </div>
    </div>

    <div v-if="result" class="result-section">
      <h3>Result</h3>
      <div :class="['result-box', result.success ? 'success' : 'error']">
        <pre>{{ result.output }}</pre>
      </div>
    </div>

    <div v-if="commandHistory.length > 0" class="history-section">
      <h3>Command History</h3>
      <div class="history-list">
        <div
          v-for="(item, index) in commandHistory"
          :key="index"
          :class="['history-item', item.success ? 'success' : 'error']"
        >
          <div class="history-header">
            <span class="timestamp">{{ formatTimestamp(item.timestamp) }}</span>
            <span class="command-name">{{ item.command }}</span>
            <span :class="['status-badge', item.success ? 'success' : 'error']">
              {{ item.success ? 'Success' : 'Failed' }}
            </span>
          </div>
          <div v-if="item.params && Object.keys(item.params).length > 0" class="history-params">
            <small>Params: {{ JSON.stringify(item.params) }}</small>
          </div>
        </div>
      </div>
    </div>

    <div class="safety-notice">
      <strong>⚠️ Safety Notice:</strong> Commands are validated and restricted to safe operations.
      Destructive commands are blocked for safety.
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { apiRequest } from '../../utils/api'

interface PredefinedCommand {
  name: string
  label: string
  action: string
  params: Array<{
    key: string
    label: string
    type?: string
    placeholder?: string
    description?: string
  }>
}

interface CommandResult {
  success: boolean
  output: string
}

interface HistoryItem {
  timestamp: Date
  command: string
  params: Record<string, string>
  success: boolean
}

const predefinedCommands = ref<PredefinedCommand[]>([])
const selectedCommand = ref('')
const customAction = ref('')
const commandParams = ref<Record<string, string>>({})
const executing = ref(false)
const result = ref<CommandResult | null>(null)
const commandHistory = ref<HistoryItem[]>([])

const currentCommandParams = computed(() => {
  if (selectedCommand.value === 'custom') {
    return []
  }
  const cmd = predefinedCommands.value.find(c => c.name === selectedCommand.value)
  return cmd?.params || []
})

const canExecute = computed(() => {
  if (selectedCommand.value === 'custom') {
    return customAction.value.trim() !== ''
  }
  return selectedCommand.value !== ''
})

const fetchPredefinedCommands = async () => {
  try {
    const response = await apiRequest<{ commands: PredefinedCommand[] }>(
      '/api/admin/ami/commands'
    )
    predefinedCommands.value = response.commands || []
  } catch (error) {
    console.error('Failed to fetch predefined commands:', error)
  }
}

const onCommandChange = () => {
  commandParams.value = {}
  customAction.value = ''
  result.value = null
}

const executeCommand = async () => {
  if (!canExecute.value) return

  executing.value = true
  result.value = null

  try {
    let action = customAction.value
    let params = { ...commandParams.value }

    if (selectedCommand.value !== 'custom') {
      const cmd = predefinedCommands.value.find(c => c.name === selectedCommand.value)
      if (cmd) {
        action = cmd.action
      }
    }

    const response = await apiRequest<CommandResult>('/api/admin/ami/execute', {
      method: 'POST',
      body: JSON.stringify({ action, params })
    })

    result.value = response

    // Add to history
    commandHistory.value.unshift({
      timestamp: new Date(),
      command: selectedCommand.value === 'custom' ? customAction.value : selectedCommand.value,
      params: { ...commandParams.value },
      success: response.success
    })

    // Keep only last 20 items
    if (commandHistory.value.length > 20) {
      commandHistory.value = commandHistory.value.slice(0, 20)
    }
  } catch (error: any) {
    result.value = {
      success: false,
      output: error.message || 'Command execution failed'
    }
  } finally {
    executing.value = false
  }
}

const clearForm = () => {
  selectedCommand.value = ''
  customAction.value = ''
  commandParams.value = {}
  result.value = null
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

onMounted(() => {
  fetchPredefinedCommands()
})
</script>

<style scoped>
.command-panel {
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

.command-section {
  background: #ffffff;
  border-radius: 8px;
  padding: 25px;
  margin-bottom: 25px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
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

.form-group input,
.form-group select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #dce1e6;
  border-radius: 6px;
  font-size: 14px;
  transition: border-color 0.2s;
}

.form-group input:focus,
.form-group select:focus {
  outline: none;
  border-color: #3498db;
}

.param-description {
  display: block;
  margin-top: 5px;
  color: #7f8c8d;
  font-size: 12px;
}

.params-section {
  margin-top: 25px;
  padding-top: 20px;
  border-top: 1px solid #ecf0f1;
}

.params-section h3 {
  margin: 0 0 15px 0;
  font-size: 16px;
  color: #2c3e50;
}

.action-buttons {
  margin-top: 25px;
  display: flex;
  gap: 10px;
}

.btn-primary,
.btn-secondary {
  padding: 10px 20px;
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

.btn-secondary {
  background: #ecf0f1;
  color: #34495e;
}

.btn-secondary:hover {
  background: #dce1e6;
}

.result-section {
  background: #ffffff;
  border-radius: 8px;
  padding: 25px;
  margin-bottom: 25px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.result-section h3 {
  margin: 0 0 15px 0;
  font-size: 16px;
  color: #2c3e50;
}

.result-box {
  padding: 15px;
  border-radius: 6px;
  font-family: 'Courier New', monospace;
  font-size: 13px;
}

.result-box.success {
  background: #d5f4e6;
  border: 1px solid #27ae60;
  color: #145a32;
}

.result-box.error {
  background: #fadbd8;
  border: 1px solid #e74c3c;
  color: #7b241c;
}

.result-box pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
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
}

.timestamp {
  font-size: 12px;
  color: #7f8c8d;
}

.command-name {
  font-weight: 600;
  font-size: 13px;
  color: #2c3e50;
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

.history-params {
  font-size: 12px;
  color: #7f8c8d;
}

.safety-notice {
  background: #fff3cd;
  border: 1px solid #ffc107;
  border-radius: 6px;
  padding: 15px;
  font-size: 13px;
  color: #856404;
}

.safety-notice strong {
  display: block;
  margin-bottom: 5px;
}
</style>
