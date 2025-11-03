<template>
  <div class="scheduled-tasks">
    <div class="panel-header">
      <h2>Scheduled Tasks</h2>
      <p class="subtitle">Automate operations with cron-based scheduling</p>
      <button @click="showCreateModal = true" class="btn-primary">
        ➕ Create Task
      </button>
    </div>

    <!-- Tasks List -->
    <div v-if="loading" class="loading">Loading scheduled tasks...</div>
    <div v-else-if="tasks.length === 0" class="empty-state">
      <p>No scheduled tasks yet. Create one to automate operations!</p>
    </div>
    <div v-else class="tasks-list">
      <div v-for="task in tasks" :key="task.id" class="task-card">
        <div class="task-header">
          <div class="task-title">
            <h3>{{ task.name }}</h3>
            <span :class="['status-badge', task.is_active ? 'active' : 'paused']">
              {{ task.is_active ? '● Active' : '⏸ Paused' }}
            </span>
          </div>
          <div class="task-actions">
            <button
              @click="toggleTaskStatus(task)"
              :class="['btn-icon', task.is_active ? 'pause' : 'play']"
              :title="task.is_active ? 'Pause' : 'Resume'"
            >
              {{ task.is_active ? '⏸' : '▶' }}
            </button>
            <button @click="viewLogs(task)" class="btn-icon" title="View Logs">📊</button>
            <button @click="editTask(task)" class="btn-icon" title="Edit">✏️</button>
            <button @click="confirmDelete(task)" class="btn-icon" title="Delete">🗑️</button>
          </div>
        </div>
        
        <div class="task-details">
          <div class="task-type">
            <span class="label">Type:</span>
            <span class="value">{{ formatTaskType(task.task_type) }}</span>
          </div>
          <div class="task-schedule">
            <span class="label">Schedule:</span>
            <span class="value">{{ task.cron_expression }}</span>
            <span class="cron-hint">({{ describeCron(task.cron_expression) }})</span>
          </div>
          <div class="task-next-run" v-if="task.next_run">
            <span class="label">Next Run:</span>
            <span class="value">{{ formatDateTime(task.next_run) }}</span>
          </div>
          <div class="task-last-run" v-if="task.last_execution_at">
            <span class="label">Last Run:</span>
            <span class="value">{{ formatDateTime(task.last_execution_at) }}</span>
            <span :class="['execution-status', task.last_execution_success ? 'success' : 'error']">
              {{ task.last_execution_success ? '✓' : '✗' }}
            </span>
          </div>
        </div>

        <div v-if="task.parameters" class="task-parameters">
          <strong>Parameters:</strong>
          <pre>{{ formatParameters(task.parameters) }}</pre>
        </div>
      </div>
    </div>

    <!-- Message Display -->
    <div v-if="message" :class="['message-box', message.type]">
      {{ message.text }}
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showCreateModal || showEditModal" class="modal-overlay" @click="closeModals">
      <div class="modal-content large" @click.stop>
        <div class="modal-header">
          <h3>{{ showEditModal ? 'Edit Task' : 'Create Task' }}</h3>
          <button @click="closeModals" class="btn-close">✕</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>Task Name:</label>
            <input
              v-model="taskForm.name"
              type="text"
              placeholder="e.g., Nightly Net Link"
              maxlength="100"
            />
          </div>
          
          <div class="form-group">
            <label>Task Type:</label>
            <select v-model="taskForm.task_type">
              <option value="link_nodes">Link Nodes</option>
              <option value="unlink_nodes">Unlink Nodes</option>
              <option value="execute_ami">Execute AMI Command</option>
              <option value="backup_db">Database Backup</option>
            </select>
          </div>

          <div class="form-group">
            <label>Cron Expression:</label>
            <input
              v-model="taskForm.cron_expression"
              type="text"
              placeholder="e.g., 0 20 * * * (daily at 8 PM)"
            />
            <small>
              Format: second minute hour day month weekday
              <br>
              Examples: "0 0 * * * *" (hourly), "0 30 9 * * *" (daily at 9:30 AM)
            </small>
          </div>

          <!-- Parameters based on task type -->
          <div v-if="taskForm.task_type === 'link_nodes' || taskForm.task_type === 'unlink_nodes'" class="form-group">
            <label>Local Node:</label>
            <input v-model="taskForm.localNode" type="number" placeholder="e.g., 12345" />
          </div>

          <div v-if="taskForm.task_type === 'link_nodes' || taskForm.task_type === 'unlink_nodes'" class="form-group">
            <label>Remote Node:</label>
            <input v-model="taskForm.remoteNode" type="number" placeholder="e.g., 54321" />
          </div>

          <div v-if="taskForm.task_type === 'link_nodes'" class="form-group">
            <label>
              <input type="checkbox" v-model="taskForm.permanent" />
              Permanent Link
            </label>
          </div>

          <div v-if="taskForm.task_type === 'execute_ami'" class="form-group">
            <label>AMI Action:</label>
            <input v-model="taskForm.amiAction" type="text" placeholder="e.g., Command" />
          </div>

          <div v-if="taskForm.task_type === 'execute_ami'" class="form-group">
            <label>AMI Parameters (JSON):</label>
            <textarea
              v-model="taskForm.amiParameters"
              placeholder='{"Node": "12345", "Command": "status"}'
              rows="4"
            ></textarea>
          </div>

          <div v-if="taskForm.task_type === 'backup_db'" class="form-group">
            <label>Backup Comment:</label>
            <input v-model="taskForm.backupComment" type="text" placeholder="Scheduled backup" />
          </div>
        </div>
        <div class="modal-footer">
          <button @click="closeModals" class="btn-secondary">Cancel</button>
          <button
            @click="showEditModal ? updateTask() : createTask()"
            :disabled="!isFormValid || saving"
            class="btn-primary"
          >
            {{ saving ? 'Saving...' : (showEditModal ? 'Update' : 'Create') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteModal" class="modal-overlay" @click="closeModals">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>Delete Task</h3>
          <button @click="closeModals" class="btn-close">✕</button>
        </div>
        <div class="modal-body">
          <p>Are you sure you want to delete the task <strong>{{ taskToDelete?.name }}</strong>?</p>
          <p>This action cannot be undone.</p>
        </div>
        <div class="modal-footer">
          <button @click="closeModals" class="btn-secondary">Cancel</button>
          <button @click="deleteTask" :disabled="deleting" class="btn-danger">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Logs Modal -->
    <div v-if="showLogsModal" class="modal-overlay" @click="closeModals">
      <div class="modal-content large" @click.stop>
        <div class="modal-header">
          <h3>Execution Logs: {{ selectedTask?.name }}</h3>
          <button @click="closeModals" class="btn-close">✕</button>
        </div>
        <div class="modal-body">
          <div v-if="loadingLogs" class="loading">Loading logs...</div>
          <div v-else-if="executionLogs.length === 0" class="empty-state">
            <p>No execution history yet.</p>
          </div>
          <div v-else class="logs-list">
            <div v-for="log in executionLogs" :key="log.id" class="log-entry">
              <div class="log-header">
                <span class="log-time">{{ formatDateTime(log.executed_at) }}</span>
                <span :class="['log-status', log.success ? 'success' : 'error']">
                  {{ log.success ? '✓ Success' : '✗ Failed' }}
                </span>
                <span class="log-duration">{{ log.duration_ms }}ms</span>
              </div>
              <div v-if="log.result" class="log-result">
                <pre>{{ log.result }}</pre>
              </div>
              <div v-if="log.error" class="log-error">
                <strong>Error:</strong> {{ log.error }}
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="closeModals" class="btn-secondary">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../../stores/auth'

export default {
  name: 'ScheduledTasks',
  setup() {
    const authStore = useAuthStore()
    const tasks = ref([])
    const executionLogs = ref([])
    const loading = ref(false)
    const loadingLogs = ref(false)
    const saving = ref(false)
    const deleting = ref(false)
    const message = ref(null)

    const showCreateModal = ref(false)
    const showEditModal = ref(false)
    const showDeleteModal = ref(false)
    const showLogsModal = ref(false)
    const taskToDelete = ref(null)
    const selectedTask = ref(null)

    const taskForm = ref({
      id: null,
      name: '',
      task_type: 'link_nodes',
      cron_expression: '',
      localNode: '',
      remoteNode: '',
      permanent: false,
      amiAction: '',
      amiParameters: '',
      backupComment: ''
    })

    const apiHeaders = computed(() => ({
      'Authorization': `Bearer ${authStore.token}`,
      'Content-Type': 'application/json'
    }))

    const isFormValid = computed(() => {
      return taskForm.value.name && taskForm.value.cron_expression
    })

    const fetchTasks = async () => {
      loading.value = true
      try {
        const response = await fetch('/api/admin/scheduled-tasks', {
          headers: apiHeaders.value
        })
        if (!response.ok) throw new Error('Failed to fetch tasks')
        const data = await response.json()
        tasks.value = data.data || []
      } catch (error) {
        showMessage('Failed to load scheduled tasks: ' + error.message, 'error')
      } finally {
        loading.value = false
      }
    }

    const createTask = async () => {
      saving.value = true
      try {
        const parameters = buildParameters()
        const response = await fetch('/api/admin/scheduled-tasks', {
          method: 'POST',
          headers: apiHeaders.value,
          body: JSON.stringify({
            name: taskForm.value.name,
            task_type: taskForm.value.task_type,
            cron_expression: taskForm.value.cron_expression,
            parameters: parameters
          })
        })
        if (!response.ok) throw new Error('Failed to create task')
        showMessage('Task created successfully', 'success')
        closeModals()
        await fetchTasks()
      } catch (error) {
        showMessage('Failed to create task: ' + error.message, 'error')
      } finally {
        saving.value = false
      }
    }

    const updateTask = async () => {
      saving.value = true
      try {
        const parameters = buildParameters()
        const response = await fetch(`/api/admin/scheduled-tasks/${taskForm.value.id}`, {
          method: 'PUT',
          headers: apiHeaders.value,
          body: JSON.stringify({
            name: taskForm.value.name,
            cron_expression: taskForm.value.cron_expression,
            parameters: parameters
          })
        })
        if (!response.ok) throw new Error('Failed to update task')
        showMessage('Task updated successfully', 'success')
        closeModals()
        await fetchTasks()
      } catch (error) {
        showMessage('Failed to update task: ' + error.message, 'error')
      } finally {
        saving.value = false
      }
    }

    const toggleTaskStatus = async (task) => {
      try {
        const response = await fetch(`/api/admin/scheduled-tasks/${task.id}`, {
          method: 'PUT',
          headers: apiHeaders.value,
          body: JSON.stringify({
            name: task.name,
            cron_expression: task.cron_expression,
            is_active: !task.is_active,
            parameters: task.parameters
          })
        })
        if (!response.ok) throw new Error('Failed to toggle task status')
        showMessage(`Task ${!task.is_active ? 'resumed' : 'paused'} successfully`, 'success')
        await fetchTasks()
      } catch (error) {
        showMessage('Failed to toggle task status: ' + error.message, 'error')
      }
    }

    const deleteTask = async () => {
      deleting.value = true
      try {
        const response = await fetch(`/api/admin/scheduled-tasks/${taskToDelete.value.id}`, {
          method: 'DELETE',
          headers: apiHeaders.value
        })
        if (!response.ok) throw new Error('Failed to delete task')
        showMessage('Task deleted successfully', 'success')
        closeModals()
        await fetchTasks()
      } catch (error) {
        showMessage('Failed to delete task: ' + error.message, 'error')
      } finally {
        deleting.value = false
      }
    }

    const viewLogs = async (task) => {
      selectedTask.value = task
      showLogsModal.value = true
      loadingLogs.value = true
      try {
        const response = await fetch(`/api/admin/scheduled-tasks/${task.id}/logs`, {
          headers: apiHeaders.value
        })
        if (!response.ok) throw new Error('Failed to fetch logs')
        const data = await response.json()
        executionLogs.value = data.data || []
      } catch (error) {
        showMessage('Failed to load execution logs: ' + error.message, 'error')
        executionLogs.value = []
      } finally {
        loadingLogs.value = false
      }
    }

    const editTask = (task) => {
      taskForm.value = {
        id: task.id,
        name: task.name,
        task_type: task.task_type,
        cron_expression: task.cron_expression,
        localNode: task.parameters?.local_node || '',
        remoteNode: task.parameters?.remote_node || '',
        permanent: task.parameters?.permanent || false,
        amiAction: task.parameters?.action || '',
        amiParameters: task.parameters?.parameters ? JSON.stringify(task.parameters.parameters, null, 2) : '',
        backupComment: task.parameters?.comment || ''
      }
      showEditModal.value = true
    }

    const confirmDelete = (task) => {
      taskToDelete.value = task
      showDeleteModal.value = true
    }

    const closeModals = () => {
      showCreateModal.value = false
      showEditModal.value = false
      showDeleteModal.value = false
      showLogsModal.value = false
      taskToDelete.value = null
      selectedTask.value = null
      executionLogs.value = []
      taskForm.value = {
        id: null,
        name: '',
        task_type: 'link_nodes',
        cron_expression: '',
        localNode: '',
        remoteNode: '',
        permanent: false,
        amiAction: '',
        amiParameters: '',
        backupComment: ''
      }
    }

    const buildParameters = () => {
      const params = {}
      if (taskForm.value.task_type === 'link_nodes' || taskForm.value.task_type === 'unlink_nodes') {
        params.local_node = parseInt(taskForm.value.localNode)
        params.remote_node = parseInt(taskForm.value.remoteNode)
        if (taskForm.value.task_type === 'link_nodes') {
          params.permanent = taskForm.value.permanent
        }
      } else if (taskForm.value.task_type === 'execute_ami') {
        params.action = taskForm.value.amiAction
        try {
          params.parameters = JSON.parse(taskForm.value.amiParameters || '{}')
        } catch (e) {
          params.parameters = {}
        }
      } else if (taskForm.value.task_type === 'backup_db') {
        params.comment = taskForm.value.backupComment || 'Scheduled backup'
      }
      return params
    }

    const formatTaskType = (type) => {
      const types = {
        link_nodes: 'Link Nodes',
        unlink_nodes: 'Unlink Nodes',
        execute_ami: 'Execute AMI',
        backup_db: 'Database Backup'
      }
      return types[type] || type
    }

    const describeCron = (cron) => {
      // Simple cron descriptions for common patterns
      if (cron === '0 0 * * * *') return 'Every hour'
      if (cron === '0 30 9 * * *') return 'Daily at 9:30 AM'
      if (cron === '0 0 0 * * *') return 'Daily at midnight'
      if (cron === '0 0 20 * * *') return 'Daily at 8 PM'
      if (cron.startsWith('0 0 0 * * ')) return 'Weekly on ' + getDayName(cron)
      return 'Custom schedule'
    }

    const getDayName = (cron) => {
      const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
      const parts = cron.split(' ')
      const day = parseInt(parts[5])
      return days[day] || ''
    }

    const formatDateTime = (dateString) => {
      if (!dateString) return 'N/A'
      const date = new Date(dateString)
      return date.toLocaleString()
    }

    const formatParameters = (params) => {
      if (!params) return 'None'
      return JSON.stringify(params, null, 2)
    }

    const showMessage = (text, type) => {
      message.value = { text, type }
      setTimeout(() => {
        message.value = null
      }, 5000)
    }

    onMounted(() => {
      fetchTasks()
    })

    return {
      tasks,
      executionLogs,
      loading,
      loadingLogs,
      saving,
      deleting,
      message,
      showCreateModal,
      showEditModal,
      showDeleteModal,
      showLogsModal,
      taskToDelete,
      selectedTask,
      taskForm,
      isFormValid,
      createTask,
      updateTask,
      toggleTaskStatus,
      deleteTask,
      viewLogs,
      editTask,
      confirmDelete,
      closeModals,
      formatTaskType,
      describeCron,
      formatDateTime,
      formatParameters
    }
  }
}
</script>

<style scoped>
.scheduled-tasks {
  padding: 2rem;
  max-width: 1400px;
  margin: 0 auto;
}

.panel-header {
  margin-bottom: 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.panel-header h2 {
  margin: 0;
  font-size: 2rem;
  color: #333;
}

.subtitle {
  color: #666;
  margin: 0.5rem 0 0 0;
}

.loading, .empty-state {
  text-align: center;
  padding: 3rem;
  color: #666;
  font-size: 1.1rem;
}

.tasks-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(450px, 1fr));
  gap: 1.5rem;
}

.task-card {
  background: white;
  border-radius: 8px;
  padding: 1.5rem;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  transition: box-shadow 0.3s;
}

.task-card:hover {
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.task-title {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex: 1;
}

.task-title h3 {
  margin: 0;
  font-size: 1.3rem;
  color: #333;
}

.status-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.85rem;
  font-weight: 500;
}

.status-badge.active {
  background: #d4edda;
  color: #155724;
}

.status-badge.paused {
  background: #f8d7da;
  color: #721c24;
}

.task-actions {
  display: flex;
  gap: 0.5rem;
}

.btn-icon {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1.2rem;
  padding: 0.25rem;
  opacity: 0.7;
  transition: opacity 0.2s;
}

.btn-icon:hover {
  opacity: 1;
}

.btn-icon.play {
  color: #28a745;
}

.btn-icon.pause {
  color: #ffc107;
}

.task-details {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.task-details > div {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.task-details .label {
  font-weight: 600;
  color: #666;
  min-width: 90px;
}

.task-details .value {
  color: #333;
  font-family: 'Courier New', monospace;
}

.cron-hint {
  color: #888;
  font-size: 0.85rem;
  font-style: italic;
}

.execution-status {
  font-size: 1.2rem;
  margin-left: 0.5rem;
}

.execution-status.success {
  color: #28a745;
}

.execution-status.error {
  color: #dc3545;
}

.task-parameters {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid #eee;
}

.task-parameters strong {
  display: block;
  margin-bottom: 0.5rem;
  color: #555;
}

.task-parameters pre {
  background: #f8f9fa;
  padding: 0.75rem;
  border-radius: 4px;
  font-size: 0.85rem;
  overflow-x: auto;
  margin: 0;
}

.message-box {
  margin-top: 1.5rem;
  padding: 1rem;
  border-radius: 4px;
  font-weight: 500;
}

.message-box.success {
  background-color: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.message-box.error {
  background-color: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 8px;
  max-width: 600px;
  width: 90%;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
}

.modal-content.large {
  max-width: 900px;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #eee;
}

.modal-header h3 {
  margin: 0;
  font-size: 1.5rem;
}

.btn-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #999;
  padding: 0;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-close:hover {
  color: #333;
}

.modal-body {
  padding: 1.5rem;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: #333;
}

.form-group input[type="text"],
.form-group input[type="number"],
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 1rem;
  box-sizing: border-box;
}

.form-group input[type="text"]:focus,
.form-group input[type="number"]:focus,
.form-group select:focus,
.form-group textarea:focus {
  outline: none;
  border-color: #007bff;
}

.form-group input[type="checkbox"] {
  margin-right: 0.5rem;
}

.form-group small {
  display: block;
  margin-top: 0.25rem;
  color: #666;
  font-size: 0.85rem;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  padding: 1.5rem;
  border-top: 1px solid #eee;
}

.btn-primary,
.btn-secondary,
.btn-danger {
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 4px;
  font-size: 1rem;
  cursor: pointer;
  font-weight: 500;
  transition: all 0.2s;
}

.btn-primary {
  background-color: #007bff;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background-color: #0056b3;
}

.btn-secondary {
  background-color: #6c757d;
  color: white;
}

.btn-secondary:hover {
  background-color: #545b62;
}

.btn-danger {
  background-color: #dc3545;
  color: white;
}

.btn-danger:hover:not(:disabled) {
  background-color: #c82333;
}

.btn-primary:disabled,
.btn-danger:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.logs-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.log-entry {
  background: #f8f9fa;
  border-radius: 4px;
  padding: 1rem;
  border-left: 4px solid #dee2e6;
}

.log-entry.success {
  border-left-color: #28a745;
}

.log-entry.error {
  border-left-color: #dc3545;
}

.log-header {
  display: flex;
  gap: 1rem;
  align-items: center;
  margin-bottom: 0.5rem;
}

.log-time {
  color: #666;
  font-size: 0.9rem;
}

.log-status {
  font-weight: 600;
  font-size: 0.9rem;
}

.log-status.success {
  color: #28a745;
}

.log-status.error {
  color: #dc3545;
}

.log-duration {
  color: #888;
  font-size: 0.85rem;
  margin-left: auto;
}

.log-result pre,
.log-error {
  font-size: 0.85rem;
  margin-top: 0.5rem;
}

.log-result pre {
  background: white;
  padding: 0.75rem;
  border-radius: 4px;
  overflow-x: auto;
}

.log-error {
  color: #dc3545;
}
</style>
