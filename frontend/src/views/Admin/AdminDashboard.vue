<template>
  <div class="admin-dashboard">
    <div class="admin-header">
      <h1>Admin Dashboard</h1>
      <p class="subtitle">Manage your Allstar Nexus system</p>
    </div>

    <div class="admin-cards">
      <!-- System Status Card -->
      <div class="admin-card">
        <div class="card-icon">🖥️</div>
        <h3>System Status</h3>
        <p>Monitor system health and resources</p>
        <div class="card-stats" v-if="systemStatus">
          <div class="stat">
            <span class="label">Uptime:</span>
            <span class="value">{{ systemStatus.uptime }}</span>
          </div>
        </div>
        <button @click="$router.push('/admin/system')" class="btn-card">View Details</button>
      </div>

      <!-- User Management Card -->
      <div class="admin-card">
        <div class="card-icon">👥</div>
        <h3>User Management</h3>
        <p>Manage users, roles, and permissions</p>
        <div class="card-stats" v-if="userCount !== null">
          <div class="stat">
            <span class="label">Total Users:</span>
            <span class="value">{{ userCount }}</span>
          </div>
        </div>
        <button @click="$router.push('/admin/users')" class="btn-card">Manage Users</button>
      </div>

      <!-- Config Editor Card -->
      <div class="admin-card">
        <div class="card-icon">⚙️</div>
        <h3>Configuration</h3>
        <p>Edit config.yaml and manage backups</p>
        <button @click="$router.push('/admin/config')" class="btn-card">Edit Config</button>
      </div>

      <!-- Command Panel Card -->
      <div class="admin-card">
        <div class="card-icon">💻</div>
        <h3>Command Panel</h3>
        <p>Execute AMI commands</p>
        <button @click="$router.push('/admin/commands')" class="btn-card">Open Panel</button>
      </div>

      <!-- Node Control Card -->
      <div class="admin-card">
        <div class="card-icon">🔗</div>
        <h3>Node Control</h3>
        <p>Link and unlink nodes via AMI</p>
        <button @click="$router.push('/admin/nodes')" class="btn-card">Manage Nodes</button>
      </div>

      <!-- Database Backup Card -->
      <div class="admin-card">
        <div class="card-icon">💾</div>
        <h3>Database Backup</h3>
        <p>Create and restore database backups</p>
        <button @click="$router.push('/admin/database')" class="btn-card">Manage Backups</button>
      </div>

      <!-- Audit Logs Card -->
      <div class="admin-card">
        <div class="card-icon">📋</div>
        <h3>Audit Logs</h3>
        <p>View administrative action history</p>
        <div class="card-stats" v-if="recentAudits.length > 0">
          <div class="stat">
            <span class="label">Recent Actions:</span>
            <span class="value">{{ recentAudits.length }}</span>
          </div>
        </div>
        <button @click="$router.push('/admin/audit')" class="btn-card">View Logs</button>
      </div>

      <!-- Log Viewer Card -->
      <div class="admin-card">
        <div class="card-icon">📄</div>
        <h3>Log Viewer</h3>
        <p>Stream and search system logs</p>
        <button @click="$router.push('/admin/logs')" class="btn-card">View Logs</button>
      </div>
    </div>

    <!-- Recent Activity -->
    <div class="recent-activity" v-if="recentAudits.length > 0">
      <h2>Recent Admin Activity</h2>
      <div class="activity-list">
        <div v-for="audit in recentAudits.slice(0, 10)" :key="audit.id" class="activity-item">
          <div class="activity-icon" :class="audit.success ? 'success' : 'error'">
            {{ audit.success ? '✓' : '✗' }}
          </div>
          <div class="activity-details">
            <div class="activity-action">{{ formatAction(audit.action) }}</div>
            <div class="activity-meta">
              <span class="user">{{ audit.user_email }}</span>
              <span class="separator">•</span>
              <span class="time">{{ formatTime(audit.timestamp) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { logger } from '../../utils/logger'

const systemStatus = ref(null)
const userCount = ref(null)
const recentAudits = ref([])

async function loadSystemStatus() {
  try {
    const response = await fetch('/api/admin/system/status', {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    const data = await response.json()
    if (data.ok !== false && data.data) {
      systemStatus.value = data.data
    }
  } catch (e) {
    logger.error('Failed to load system status', e)
  }
}

async function loadUsers() {
  try {
    const response = await fetch('/api/admin/users', {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    const data = await response.json()
    if (data.ok !== false && data.data) {
      userCount.value = data.data.length
    }
  } catch (e) {
    logger.error('Failed to load users', e)
  }
}

async function loadRecentAudits() {
  try {
    const response = await fetch('/api/admin/audit/recent?limit=10', {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    const data = await response.json()
    if (data.ok !== false && data.data) {
      recentAudits.value = data.data
    }
  } catch (e) {
    logger.error('Failed to load audit logs', e)
  }
}

function formatAction(action) {
  const map = {
    'config.read': 'Viewed configuration',
    'config.update': 'Updated configuration',
    'config.backup': 'Created config backup',
    'config.restore': 'Restored configuration',
    'command.execute': 'Executed command',
    'node.link': 'Linked node',
    'node.unlink': 'Unlinked node',
    'user.create': 'Created user',
    'user.update': 'Updated user',
    'user.delete': 'Deleted user',
    'user.password_reset': 'Reset user password',
    'auth.login': 'Logged in',
    'auth.login_failed': 'Login failed'
  }
  return map[action] || action
}

function formatTime(timestamp) {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now - date

  if (diff < 60000) return 'just now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
  return date.toLocaleDateString()
}

onMounted(() => {
  loadSystemStatus()
  loadUsers()
  loadRecentAudits()
})
</script>

<style scoped>
.admin-dashboard {
  padding: 2rem;
  max-width: 1400px;
  margin: 0 auto;
}

.admin-header {
  margin-bottom: 2rem;
}

.admin-header h1 {
  margin: 0 0 0.5rem 0;
  color: var(--text-primary);
}

.subtitle {
  margin: 0;
  color: var(--text-muted);
  font-size: 1.1rem;
}

.admin-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
  margin-bottom: 3rem;
}

.admin-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 2rem;
  transition: all 0.3s ease;
}

.admin-card:hover {
  border-color: var(--accent-primary);
  box-shadow: 0 4px 16px var(--shadow);
  transform: translateY(-2px);
}

.card-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
}

.admin-card h3 {
  margin: 0 0 0.5rem 0;
  color: var(--text-primary);
}

.admin-card p {
  margin: 0 0 1rem 0;
  color: var(--text-muted);
  min-height: 3rem;
}

.card-stats {
  margin: 1rem 0;
  padding: 1rem;
  background: var(--bg-tertiary);
  border-radius: 8px;
}

.stat {
  display: flex;
  justify-content: space-between;
  margin: 0.5rem 0;
}

.stat .label {
  color: var(--text-muted);
}

.stat .value {
  color: var(--text-primary);
  font-weight: 600;
}

.btn-card {
  width: 100%;
  padding: 0.75rem;
  background: linear-gradient(135deg, var(--accent-gradient-start), var(--accent-gradient-end));
  color: #fff;
  border: none;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px var(--shadow);
}

.recent-activity {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 2rem;
}

.recent-activity h2 {
  margin: 0 0 1.5rem 0;
  color: var(--text-primary);
}

.activity-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.activity-item {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  background: var(--bg-tertiary);
  border-radius: 8px;
  transition: background 0.2s;
}

.activity-item:hover {
  background: var(--bg-hover);
}

.activity-icon {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  flex-shrink: 0;
}

.activity-icon.success {
  background: color-mix(in srgb, var(--success) 20%, var(--bg-secondary));
  color: var(--success);
}

.activity-icon.error {
  background: color-mix(in srgb, var(--error) 20%, var(--bg-secondary));
  color: var(--error);
}

.activity-details {
  flex: 1;
}

.activity-action {
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.25rem;
}

.activity-meta {
  font-size: 0.875rem;
  color: var(--text-muted);
}

.activity-meta .separator {
  margin: 0 0.5rem;
}

@media (max-width: 768px) {
  .admin-dashboard {
    padding: 1rem;
  }

  .admin-cards {
    grid-template-columns: 1fr;
  }
}
</style>
