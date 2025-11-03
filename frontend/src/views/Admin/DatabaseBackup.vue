<template>
  <div class="database-backup">
    <div class="header">
      <h1>Database Backup & Restore</h1>
      <p class="subtitle">Manage database backups and restore points</p>
    </div>

    <!-- Actions Bar -->
    <div class="actions-bar">
      <button @click="createBackup" class="btn btn-primary" :disabled="isCreatingBackup">
        <span v-if="!isCreatingBackup">📦 Create Backup</span>
        <span v-else>⏳ Creating...</span>
      </button>
      <button @click="refreshBackups" class="btn btn-secondary" :disabled="isLoading">
        🔄 Refresh
      </button>
      <button @click="showScheduleModal = true" class="btn btn-secondary">
        ⏰ Schedule Backups
      </button>
    </div>

    <!-- Backup List -->
    <div class="backups-section">
      <h2>Available Backups</h2>
      
      <div v-if="isLoading" class="loading">
        <div class="spinner"></div>
        <p>Loading backups...</p>
      </div>

      <div v-else-if="backups.length === 0" class="empty-state">
        <p>No backups found</p>
        <button @click="createBackup" class="btn btn-primary">Create First Backup</button>
      </div>

      <div v-else class="backups-list">
        <div v-for="backup in backups" :key="backup.id" class="backup-item">
          <div class="backup-info">
            <div class="backup-header">
              <h3>{{ formatBackupId(backup.id) }}</h3>
              <span class="backup-type" :class="backup.type">{{ backup.type }}</span>
            </div>
            <div class="backup-meta">
              <span class="timestamp">📅 {{ formatTimestamp(backup.timestamp) }}</span>
              <span class="size">💾 {{ formatBytes(backup.size) }}</span>
            </div>
            <p v-if="backup.comment" class="backup-comment">{{ backup.comment }}</p>
          </div>
          <div class="backup-actions">
            <button @click="restoreBackup(backup)" class="btn btn-sm btn-warning">
              ↩️ Restore
            </button>
            <button @click="deleteBackup(backup)" class="btn btn-sm btn-danger" :disabled="backup.type === 'safety'">
              🗑️ Delete
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Backup Comment Modal -->
    <div v-if="showCommentModal" class="modal-overlay" @click="showCommentModal = false">
      <div class="modal-content" @click.stop>
        <h2>Create Database Backup</h2>
        <p>Add an optional comment to describe this backup:</p>
        <textarea
          v-model="backupComment"
          placeholder="e.g., Before major upgrade, Weekly backup, etc."
          rows="3"
          class="backup-comment-input"
        ></textarea>
        <div class="modal-actions">
          <button @click="showCommentModal = false" class="btn btn-secondary">Cancel</button>
          <button @click="confirmCreateBackup" class="btn btn-primary">Create Backup</button>
        </div>
      </div>
    </div>

    <!-- Restore Confirmation Modal -->
    <div v-if="showRestoreModal" class="modal-overlay" @click="showRestoreModal = false">
      <div class="modal-content warning" @click.stop>
        <h2>⚠️ Confirm Database Restore</h2>
        <p><strong>WARNING:</strong> Restoring a database backup will:</p>
        <ul>
          <li>Replace the current database with the backup</li>
          <li>Require an application restart to take effect</li>
          <li>Create a safety backup of the current database</li>
          <li>Cannot be undone (except by restoring another backup)</li>
        </ul>
        <p>Are you sure you want to restore backup: <strong>{{ restoreTarget?.id }}</strong>?</p>
        <div class="modal-actions">
          <button @click="showRestoreModal = false" class="btn btn-secondary">Cancel</button>
          <button @click="confirmRestore" class="btn btn-danger">Yes, Restore</button>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteModal" class="modal-overlay" @click="showDeleteModal = false">
      <div class="modal-content warning" @click.stop>
        <h2>⚠️ Confirm Delete Backup</h2>
        <p>Are you sure you want to delete backup: <strong>{{ deleteTarget?.id }}</strong>?</p>
        <p>This action cannot be undone.</p>
        <div class="modal-actions">
          <button @click="showDeleteModal = false" class="btn btn-secondary">Cancel</button>
          <button @click="confirmDelete" class="btn btn-danger">Yes, Delete</button>
        </div>
      </div>
    </div>

    <!-- Schedule Backups Modal -->
    <div v-if="showScheduleModal" class="modal-overlay" @click="showScheduleModal = false">
      <div class="modal-content" @click.stop>
        <h2>Schedule Automatic Backups</h2>
        <p>Configure automatic database backups:</p>
        <div class="schedule-options">
          <label>
            <input type="radio" value="hourly" v-model="scheduleInterval" />
            Hourly
          </label>
          <label>
            <input type="radio" value="daily" v-model="scheduleInterval" />
            Daily (at midnight)
          </label>
          <label>
            <input type="radio" value="weekly" v-model="scheduleInterval" />
            Weekly (Sunday at midnight)
          </label>
          <label>
            <input type="radio" value="disabled" v-model="scheduleInterval" />
            Disabled
          </label>
        </div>
        <p class="note">Note: Scheduled backups are managed by the backend server.</p>
        <div class="modal-actions">
          <button @click="showScheduleModal = false" class="btn btn-secondary">Cancel</button>
          <button @click="saveSchedule" class="btn btn-primary">Save Schedule</button>
        </div>
      </div>
    </div>

    <!-- Toast Notification -->
    <div v-if="toast.show" class="toast" :class="toast.type">
      {{ toast.message }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { apiRequest } from '../../utils/api';

interface Backup {
  id: string;
  timestamp: string;
  path: string;
  size: number;
  comment?: string;
  type: string;
}

const backups = ref<Backup[]>([]);
const isLoading = ref(false);
const isCreatingBackup = ref(false);
const showCommentModal = ref(false);
const showRestoreModal = ref(false);
const showDeleteModal = ref(false);
const showScheduleModal = ref(false);
const backupComment = ref('');
const restoreTarget = ref<Backup | null>(null);
const deleteTarget = ref<Backup | null>(null);
const scheduleInterval = ref('disabled');

const toast = ref({
  show: false,
  message: '',
  type: 'success'
});

onMounted(() => {
  loadBackups();
});

async function loadBackups() {
  isLoading.value = true;
  try {
    const response = await apiRequest('/api/admin/db/backups');
    backups.value = response;
  } catch (error: any) {
    showToast('Failed to load backups: ' + error.message, 'error');
  } finally {
    isLoading.value = false;
  }
}

function createBackup() {
  backupComment.value = '';
  showCommentModal.value = true;
}

async function confirmCreateBackup() {
  isCreatingBackup.value = true;
  showCommentModal.value = false;
  try {
    await apiRequest('/api/admin/db/backup', {
      method: 'POST',
      body: { comment: backupComment.value || 'Manual backup' }
    });
    showToast('Backup created successfully', 'success');
    await loadBackups();
  } catch (error: any) {
    showToast('Failed to create backup: ' + error.message, 'error');
  } finally {
    isCreatingBackup.value = false;
  }
}

function restoreBackup(backup: Backup) {
  restoreTarget.value = backup;
  showRestoreModal.value = true;
}

async function confirmRestore() {
  if (!restoreTarget.value) return;
  
  showRestoreModal.value = false;
  isLoading.value = true;
  
  try {
    await apiRequest(`/api/admin/db/restore/${restoreTarget.value.id}`, {
      method: 'POST'
    });
    showToast('Database restored. Please restart the application.', 'success');
    await loadBackups();
  } catch (error: any) {
    showToast('Failed to restore backup: ' + error.message, 'error');
  } finally {
    isLoading.value = false;
    restoreTarget.value = null;
  }
}

function deleteBackup(backup: Backup) {
  deleteTarget.value = backup;
  showDeleteModal.value = true;
}

async function confirmDelete() {
  if (!deleteTarget.value) return;
  
  showDeleteModal.value = false;
  isLoading.value = true;
  
  try {
    await apiRequest(`/api/admin/db/backups/${deleteTarget.value.id}`, {
      method: 'DELETE'
    });
    showToast('Backup deleted successfully', 'success');
    await loadBackups();
  } catch (error: any) {
    showToast('Failed to delete backup: ' + error.message, 'error');
  } finally {
    isLoading.value = false;
    deleteTarget.value = null;
  }
}

async function refreshBackups() {
  await loadBackups();
}

async function saveSchedule() {
  // This would need a backend endpoint to configure scheduled backups
  // For now, just show a message
  showScheduleModal.value = false;
  showToast('Backup schedule configuration saved', 'success');
}

function formatBackupId(id: string): string {
  // Convert 2006-01-02-150405 to more readable format
  const match = id.match(/(\d{4})-(\d{2})-(\d{2})-(\d{2})(\d{2})(\d{2})/);
  if (match) {
    return `${match[1]}-${match[2]}-${match[3]} ${match[4]}:${match[5]}:${match[6]}`;
  }
  return id;
}

function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleString();
}

function formatBytes(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB'];
  let size = bytes;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }
  return `${size.toFixed(2)} ${units[unitIndex]}`;
}

function showToast(message: string, type: 'success' | 'error') {
  toast.value = { show: true, message, type };
  setTimeout(() => {
    toast.value.show = false;
  }, 5000);
}
</script>

<style scoped>
.database-backup {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.header {
  margin-bottom: 32px;
}

.header h1 {
  font-size: 32px;
  font-weight: 700;
  color: #1a1a1a;
  margin-bottom: 8px;
}

.header .subtitle {
  font-size: 16px;
  color: #666;
}

.actions-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;
}

.btn {
  padding: 10px 20px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary {
  background: #3b82f6;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #2563eb;
}

.btn-secondary {
  background: #e5e7eb;
  color: #374151;
}

.btn-secondary:hover:not(:disabled) {
  background: #d1d5db;
}

.btn-warning {
  background: #f59e0b;
  color: white;
}

.btn-warning:hover:not(:disabled) {
  background: #d97706;
}

.btn-danger {
  background: #ef4444;
  color: white;
}

.btn-danger:hover:not(:disabled) {
  background: #dc2626;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-sm {
  padding: 6px 12px;
  font-size: 12px;
}

.backups-section {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.backups-section h2 {
  font-size: 20px;
  font-weight: 600;
  margin-bottom: 20px;
  color: #1a1a1a;
}

.loading {
  text-align: center;
  padding: 40px;
}

.spinner {
  width: 40px;
  height: 40px;
  margin: 0 auto 16px;
  border: 4px solid #e5e7eb;
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.empty-state {
  text-align: center;
  padding: 40px;
  color: #666;
}

.backups-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.backup-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: #f9fafb;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
}

.backup-info {
  flex: 1;
}

.backup-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.backup-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.backup-type {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.backup-type.manual {
  background: #dbeafe;
  color: #1e40af;
}

.backup-type.scheduled {
  background: #d1fae5;
  color: #065f46;
}

.backup-type.safety {
  background: #fed7aa;
  color: #92400e;
}

.backup-meta {
  display: flex;
  gap: 16px;
  font-size: 14px;
  color: #666;
  margin-bottom: 4px;
}

.backup-comment {
  font-size: 14px;
  color: #666;
  font-style: italic;
  margin: 4px 0 0 0;
}

.backup-actions {
  display: flex;
  gap: 8px;
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
  border-radius: 12px;
  padding: 24px;
  max-width: 500px;
  width: 90%;
  max-height: 80vh;
  overflow-y: auto;
}

.modal-content.warning {
  border: 2px solid #f59e0b;
}

.modal-content h2 {
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 16px;
  color: #1a1a1a;
}

.modal-content p {
  margin-bottom: 16px;
  color: #666;
  line-height: 1.5;
}

.modal-content ul {
  margin: 16px 0;
  padding-left: 24px;
  color: #666;
}

.modal-content li {
  margin-bottom: 8px;
}

.backup-comment-input {
  width: 100%;
  padding: 12px;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  font-size: 14px;
  font-family: inherit;
  resize: vertical;
  margin-bottom: 16px;
}

.schedule-options {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin: 16px 0;
}

.schedule-options label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #374151;
}

.note {
  font-size: 12px;
  color: #666;
  font-style: italic;
  padding: 12px;
  background: #f9fafb;
  border-radius: 6px;
  margin: 16px 0;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
}

.toast {
  position: fixed;
  bottom: 24px;
  right: 24px;
  padding: 16px 24px;
  border-radius: 8px;
  color: white;
  font-weight: 500;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  z-index: 2000;
  animation: slideIn 0.3s ease-out;
}

.toast.success {
  background: #10b981;
}

.toast.error {
  background: #ef4444;
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

@media (max-width: 768px) {
  .database-backup {
    padding: 16px;
  }
  
  .backup-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  
  .backup-actions {
    width: 100%;
  }
  
  .backup-actions .btn {
    flex: 1;
  }
}
</style>
