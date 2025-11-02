<template>
  <div class="user-manager">
    <div class="panel-header">
      <h2>User Management</h2>
      <p class="subtitle">Manage user accounts and roles</p>
      <button @click="showCreateModal = true" class="btn-primary">
        + Create New User
      </button>
    </div>

    <div v-if="loading" class="loading">Loading users...</div>

    <div v-else-if="error" class="error-box">
      {{ error }}
    </div>

    <div v-else class="users-table-container">
      <table class="users-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Email</th>
            <th>Role</th>
            <th>Created</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td>{{ user.id }}</td>
            <td>{{ user.email }}</td>
            <td>
              <span :class="['role-badge', user.role]">
                {{ user.role }}
              </span>
            </td>
            <td>{{ formatDate(user.created_at) }}</td>
            <td>
              <div class="action-buttons">
                <button @click="editUser(user)" class="btn-edit">Edit</button>
                <button @click="confirmDelete(user)" class="btn-delete">Delete</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="users.length === 0" class="no-users">
        No users found. Create your first user to get started.
      </div>
    </div>

    <!-- Create/Edit User Modal -->
    <div v-if="showCreateModal || showEditModal" class="modal-overlay" @click.self="closeModals">
      <div class="modal">
        <div class="modal-header">
          <h3>{{ showEditModal ? 'Edit User' : 'Create New User' }}</h3>
          <button @click="closeModals" class="close-btn">&times;</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label for="user-email">Email:</label>
            <input
              id="user-email"
              v-model="userForm.email"
              type="email"
              placeholder="user@example.com"
              :disabled="showEditModal"
            />
          </div>
          <div class="form-group">
            <label for="user-password">Password:</label>
            <input
              id="user-password"
              v-model="userForm.password"
              type="password"
              :placeholder="showEditModal ? 'Leave blank to keep current password' : 'Enter password'"
            />
          </div>
          <div class="form-group">
            <label for="user-role">Role:</label>
            <select id="user-role" v-model="userForm.role">
              <option value="user">User</option>
              <option value="admin">Admin</option>
              <option value="superadmin">Superadmin</option>
            </select>
          </div>
          <div v-if="modalError" class="modal-error">
            {{ modalError }}
          </div>
        </div>
        <div class="modal-footer">
          <button @click="closeModals" class="btn-secondary">Cancel</button>
          <button
            @click="showEditModal ? updateUser() : createUser()"
            :disabled="saving"
            class="btn-primary"
          >
            {{ saving ? 'Saving...' : (showEditModal ? 'Update' : 'Create') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteModal" class="modal-overlay" @click.self="closeModals">
      <div class="modal">
        <div class="modal-header">
          <h3>Confirm Delete</h3>
          <button @click="closeModals" class="close-btn">&times;</button>
        </div>
        <div class="modal-body">
          <p>Are you sure you want to delete user <strong>{{ userToDelete?.email }}</strong>?</p>
          <p class="warning">This action cannot be undone.</p>
        </div>
        <div class="modal-footer">
          <button @click="closeModals" class="btn-secondary">Cancel</button>
          <button @click="deleteUser" :disabled="deleting" class="btn-danger">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { apiRequest } from '../../utils/api'

interface User {
  id: number
  email: string
  role: string
  created_at: string
}

interface UserForm {
  email: string
  password: string
  role: string
}

const users = ref<User[]>([])
const loading = ref(false)
const error = ref('')
const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteModal = ref(false)
const userForm = ref<UserForm>({
  email: '',
  password: '',
  role: 'user'
})
const editingUserId = ref<number | null>(null)
const userToDelete = ref<User | null>(null)
const saving = ref(false)
const deleting = ref(false)
const modalError = ref('')

const fetchUsers = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await apiRequest<{ users: User[] }>('/api/admin/users')
    users.value = response.users || []
  } catch (err: any) {
    error.value = err.message || 'Failed to fetch users'
  } finally {
    loading.value = false
  }
}

const createUser = async () => {
  if (!userForm.value.email || !userForm.value.password) {
    modalError.value = 'Email and password are required'
    return
  }

  saving.value = true
  modalError.value = ''

  try {
    await apiRequest('/api/admin/users/', {
      method: 'POST',
      body: JSON.stringify({
        email: userForm.value.email,
        password: userForm.value.password,
        role: userForm.value.role
      })
    })

    closeModals()
    await fetchUsers()
  } catch (err: any) {
    modalError.value = err.message || 'Failed to create user'
  } finally {
    saving.value = false
  }
}

const editUser = (user: User) => {
  editingUserId.value = user.id
  userForm.value = {
    email: user.email,
    password: '',
    role: user.role
  }
  showEditModal.value = true
}

const updateUser = async () => {
  if (!userForm.value.email) {
    modalError.value = 'Email is required'
    return
  }

  saving.value = true
  modalError.value = ''

  try {
    const payload: any = {
      email: userForm.value.email,
      role: userForm.value.role
    }

    // Only include password if it's been changed
    if (userForm.value.password) {
      payload.password = userForm.value.password
    }

    await apiRequest(`/api/admin/users/${editingUserId.value}`, {
      method: 'PUT',
      body: JSON.stringify(payload)
    })

    closeModals()
    await fetchUsers()
  } catch (err: any) {
    modalError.value = err.message || 'Failed to update user'
  } finally {
    saving.value = false
  }
}

const confirmDelete = (user: User) => {
  userToDelete.value = user
  showDeleteModal.value = true
}

const deleteUser = async () => {
  if (!userToDelete.value) return

  deleting.value = true

  try {
    await apiRequest(`/api/admin/users/${userToDelete.value.id}`, {
      method: 'DELETE'
    })

    closeModals()
    await fetchUsers()
  } catch (err: any) {
    error.value = err.message || 'Failed to delete user'
  } finally {
    deleting.value = false
  }
}

const closeModals = () => {
  showCreateModal.value = false
  showEditModal.value = false
  showDeleteModal.value = false
  userForm.value = {
    email: '',
    password: '',
    role: 'user'
  }
  editingUserId.value = null
  userToDelete.value = null
  modalError.value = ''
}

const formatDate = (dateStr: string) => {
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(dateStr))
}

onMounted(() => {
  fetchUsers()
})
</script>

<style scoped>
.user-manager {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
  flex-wrap: wrap;
  gap: 15px;
}

.panel-header h2 {
  margin: 0;
  font-size: 28px;
  color: #2c3e50;
}

.subtitle {
  flex-basis: 100%;
  margin: 0;
  color: #7f8c8d;
  font-size: 14px;
}

.btn-primary {
  padding: 10px 20px;
  background: #3498db;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary:hover {
  background: #2980b9;
}

.loading {
  text-align: center;
  padding: 40px;
  color: #7f8c8d;
}

.error-box {
  background: #fadbd8;
  border: 1px solid #e74c3c;
  border-radius: 6px;
  padding: 15px;
  color: #7b241c;
}

.users-table-container {
  background: #ffffff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  overflow: hidden;
}

.users-table {
  width: 100%;
  border-collapse: collapse;
}

.users-table thead {
  background: #f8f9fa;
}

.users-table th {
  padding: 15px;
  text-align: left;
  font-weight: 600;
  color: #2c3e50;
  font-size: 13px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.users-table td {
  padding: 15px;
  border-top: 1px solid #ecf0f1;
  color: #34495e;
  font-size: 14px;
}

.role-badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
}

.role-badge.user {
  background: #e8f4f8;
  color: #3498db;
}

.role-badge.admin {
  background: #fff3cd;
  color: #856404;
}

.role-badge.superadmin {
  background: #fadbd8;
  color: #e74c3c;
}

.action-buttons {
  display: flex;
  gap: 8px;
}

.btn-edit,
.btn-delete {
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-edit {
  background: #3498db;
  color: white;
}

.btn-edit:hover {
  background: #2980b9;
}

.btn-delete {
  background: #e74c3c;
  color: white;
}

.btn-delete:hover {
  background: #c0392b;
}

.no-users {
  padding: 40px;
  text-align: center;
  color: #7f8c8d;
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

.modal {
  background: white;
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  box-shadow: 0 4px 20px rgba(0,0,0,0.2);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #ecf0f1;
}

.modal-header h3 {
  margin: 0;
  font-size: 20px;
  color: #2c3e50;
}

.close-btn {
  background: none;
  border: none;
  font-size: 28px;
  color: #95a5a6;
  cursor: pointer;
  line-height: 1;
  padding: 0;
  width: 30px;
  height: 30px;
}

.close-btn:hover {
  color: #2c3e50;
}

.modal-body {
  padding: 20px;
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

.form-group input:disabled {
  background: #f8f9fa;
  cursor: not-allowed;
}

.modal-error {
  background: #fadbd8;
  border: 1px solid #e74c3c;
  border-radius: 6px;
  padding: 10px;
  color: #7b241c;
  font-size: 13px;
  margin-top: 15px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 20px;
  border-top: 1px solid #ecf0f1;
}

.btn-secondary {
  padding: 10px 20px;
  background: #ecf0f1;
  color: #34495e;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-secondary:hover {
  background: #dce1e6;
}

.btn-danger {
  padding: 10px 20px;
  background: #e74c3c;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-danger:hover:not(:disabled) {
  background: #c0392b;
}

.btn-danger:disabled,
.btn-primary:disabled {
  background: #95a5a6;
  cursor: not-allowed;
}

.warning {
  color: #e74c3c;
  font-weight: 600;
  margin-top: 10px;
}
</style>
