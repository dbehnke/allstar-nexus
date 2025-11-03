<template>
  <div class="node-groups">
    <div class="panel-header">
      <h2>Node Groups</h2>
      <p class="subtitle">Organize nodes into logical collections</p>
      <button @click="showCreateModal = true" class="btn-primary">
        ➕ Create Group
      </button>
    </div>

    <!-- Groups List -->
    <div v-if="loading" class="loading">Loading node groups...</div>
    <div v-else-if="groups.length === 0" class="empty-state">
      <p>No node groups yet. Create one to get started!</p>
    </div>
    <div v-else class="groups-list">
      <div v-for="group in groups" :key="group.id" class="group-card">
        <div class="group-header">
          <h3>{{ group.name }}</h3>
          <div class="group-actions">
            <button @click="editGroup(group)" class="btn-icon" title="Edit">✏️</button>
            <button @click="confirmDelete(group)" class="btn-icon" title="Delete">🗑️</button>
          </div>
        </div>
        <p class="group-description">{{ group.description || 'No description' }}</p>
        <div class="group-stats">
          <div class="stat">
            <span class="label">Nodes:</span>
            <span class="value">{{ group.node_ids ? group.node_ids.length : 0 }}</span>
          </div>
          <div class="stat">
            <span class="label">Created:</span>
            <span class="value">{{ formatDate(group.created_at) }}</span>
          </div>
        </div>
        <div v-if="group.node_ids && group.node_ids.length > 0" class="node-list">
          <strong>Nodes:</strong>
          <span v-for="nodeId in group.node_ids" :key="nodeId" class="node-badge">
            {{ nodeId }}
          </span>
        </div>
      </div>
    </div>

    <!-- Message Display -->
    <div v-if="message" :class="['message-box', message.type]">
      {{ message.text }}
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showCreateModal || showEditModal" class="modal-overlay" @click="closeModals">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>{{ showEditModal ? 'Edit Group' : 'Create Group' }}</h3>
          <button @click="closeModals" class="btn-close">✕</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>Group Name:</label>
            <input
              v-model="groupForm.name"
              type="text"
              placeholder="e.g., Regional Net"
              maxlength="100"
            />
          </div>
          <div class="form-group">
            <label>Description:</label>
            <textarea
              v-model="groupForm.description"
              placeholder="Describe the purpose of this group..."
              rows="3"
            ></textarea>
          </div>
          <div class="form-group">
            <label>Node IDs (comma-separated):</label>
            <input
              v-model="groupForm.nodeIdsInput"
              type="text"
              placeholder="e.g., 12345, 54321, 99999"
            />
            <small>Enter node numbers separated by commas</small>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="closeModals" class="btn-secondary">Cancel</button>
          <button
            @click="showEditModal ? updateGroup() : createGroup()"
            :disabled="!groupForm.name || saving"
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
          <h3>Delete Group</h3>
          <button @click="closeModals" class="btn-close">✕</button>
        </div>
        <div class="modal-body">
          <p>Are you sure you want to delete the group <strong>{{ groupToDelete?.name }}</strong>?</p>
          <p>This action cannot be undone.</p>
        </div>
        <div class="modal-footer">
          <button @click="closeModals" class="btn-secondary">Cancel</button>
          <button @click="deleteGroup" :disabled="deleting" class="btn-danger">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../../stores/auth'

export default {
  name: 'NodeGroups',
  setup() {
    const authStore = useAuthStore()
    const groups = ref([])
    const loading = ref(false)
    const saving = ref(false)
    const deleting = ref(false)
    const message = ref(null)

    const showCreateModal = ref(false)
    const showEditModal = ref(false)
    const showDeleteModal = ref(false)
    const groupToDelete = ref(null)

    const groupForm = ref({
      id: null,
      name: '',
      description: '',
      nodeIdsInput: ''
    })

    const apiHeaders = computed(() => ({
      'Authorization': `Bearer ${authStore.token}`,
      'Content-Type': 'application/json'
    }))

    const fetchGroups = async () => {
      loading.value = true
      try {
        const response = await fetch('/api/admin/node-groups', {
          headers: apiHeaders.value
        })
        if (!response.ok) throw new Error('Failed to fetch groups')
        const data = await response.json()
        groups.value = data.data || []
      } catch (error) {
        showMessage('Failed to load node groups: ' + error.message, 'error')
      } finally {
        loading.value = false
      }
    }

    const createGroup = async () => {
      saving.value = true
      try {
        const nodeIds = parseNodeIds(groupForm.value.nodeIdsInput)
        const response = await fetch('/api/admin/node-groups', {
          method: 'POST',
          headers: apiHeaders.value,
          body: JSON.stringify({
            name: groupForm.value.name,
            description: groupForm.value.description,
            node_ids: nodeIds
          })
        })
        if (!response.ok) throw new Error('Failed to create group')
        showMessage('Group created successfully', 'success')
        closeModals()
        await fetchGroups()
      } catch (error) {
        showMessage('Failed to create group: ' + error.message, 'error')
      } finally {
        saving.value = false
      }
    }

    const updateGroup = async () => {
      saving.value = true
      try {
        const nodeIds = parseNodeIds(groupForm.value.nodeIdsInput)
        const response = await fetch(`/api/admin/node-groups/${groupForm.value.id}`, {
          method: 'PUT',
          headers: apiHeaders.value,
          body: JSON.stringify({
            name: groupForm.value.name,
            description: groupForm.value.description,
            node_ids: nodeIds
          })
        })
        if (!response.ok) throw new Error('Failed to update group')
        showMessage('Group updated successfully', 'success')
        closeModals()
        await fetchGroups()
      } catch (error) {
        showMessage('Failed to update group: ' + error.message, 'error')
      } finally {
        saving.value = false
      }
    }

    const deleteGroup = async () => {
      deleting.value = true
      try {
        const response = await fetch(`/api/admin/node-groups/${groupToDelete.value.id}`, {
          method: 'DELETE',
          headers: apiHeaders.value
        })
        if (!response.ok) throw new Error('Failed to delete group')
        showMessage('Group deleted successfully', 'success')
        closeModals()
        await fetchGroups()
      } catch (error) {
        showMessage('Failed to delete group: ' + error.message, 'error')
      } finally {
        deleting.value = false
      }
    }

    const editGroup = (group) => {
      groupForm.value = {
        id: group.id,
        name: group.name,
        description: group.description || '',
        nodeIdsInput: group.node_ids ? group.node_ids.join(', ') : ''
      }
      showEditModal.value = true
    }

    const confirmDelete = (group) => {
      groupToDelete.value = group
      showDeleteModal.value = true
    }

    const closeModals = () => {
      showCreateModal.value = false
      showEditModal.value = false
      showDeleteModal.value = false
      groupToDelete.value = null
      groupForm.value = {
        id: null,
        name: '',
        description: '',
        nodeIdsInput: ''
      }
    }

    const parseNodeIds = (input) => {
      if (!input) return []
      return input
        .split(',')
        .map(id => parseInt(id.trim()))
        .filter(id => !isNaN(id) && id > 0)
    }

    const showMessage = (text, type) => {
      message.value = { text, type }
      setTimeout(() => {
        message.value = null
      }, 5000)
    }

    const formatDate = (dateString) => {
      if (!dateString) return 'N/A'
      const date = new Date(dateString)
      return date.toLocaleDateString() + ' ' + date.toLocaleTimeString()
    }

    onMounted(() => {
      fetchGroups()
    })

    return {
      groups,
      loading,
      saving,
      deleting,
      message,
      showCreateModal,
      showEditModal,
      showDeleteModal,
      groupToDelete,
      groupForm,
      createGroup,
      updateGroup,
      deleteGroup,
      editGroup,
      confirmDelete,
      closeModals,
      formatDate
    }
  }
}
</script>

<style scoped>
.node-groups {
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

.groups-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 1.5rem;
}

.group-card {
  background: white;
  border-radius: 8px;
  padding: 1.5rem;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  transition: box-shadow 0.3s;
}

.group-card:hover {
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
}

.group-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.group-header h3 {
  margin: 0;
  font-size: 1.3rem;
  color: #333;
}

.group-actions {
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

.group-description {
  color: #666;
  margin: 0 0 1rem 0;
  font-size: 0.95rem;
}

.group-stats {
  display: flex;
  gap: 1.5rem;
  margin-bottom: 1rem;
  padding-top: 1rem;
  border-top: 1px solid #eee;
}

.stat {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.stat .label {
  font-size: 0.85rem;
  color: #888;
}

.stat .value {
  font-size: 1.1rem;
  font-weight: 600;
  color: #333;
}

.node-list {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid #eee;
}

.node-list strong {
  display: block;
  margin-bottom: 0.5rem;
  color: #555;
  font-size: 0.9rem;
}

.node-badge {
  display: inline-block;
  background: #e3f2fd;
  color: #1976d2;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  margin: 0.25rem 0.25rem 0.25rem 0;
  font-size: 0.85rem;
  font-weight: 500;
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

.form-group input,
.form-group textarea {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 1rem;
  box-sizing: border-box;
}

.form-group input:focus,
.form-group textarea:focus {
  outline: none;
  border-color: #007bff;
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
</style>
