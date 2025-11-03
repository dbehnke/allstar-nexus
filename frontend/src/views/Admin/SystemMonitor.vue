<template>
  <div class="system-monitor">
    <div class="header">
      <h1>System Health Monitoring</h1>
      <p class="subtitle">Real-time system metrics and performance data</p>
    </div>

    <!-- Health Status Banner -->
    <div v-if="health" class="health-banner" :class="health.status">
      <div class="status-icon">
        <span v-if="health.status === 'healthy'">✅</span>
        <span v-else-if="health.status === 'degraded'">⚠️</span>
        <span v-else>❌</span>
      </div>
      <div class="status-details">
        <h2>System Status: {{ health.status.toUpperCase() }}</h2>
        <p>Last checked: {{ formatTime(health.timestamp) }}</p>
      </div>
      <button @click="refreshData" class="btn-refresh" :disabled="isLoading">
        🔄 Refresh
      </button>
    </div>

    <!-- Metrics Cards -->
    <div v-if="metrics" class="metrics-grid">
      <!-- Uptime Card -->
      <div class="metric-card">
        <div class="card-icon">⏱️</div>
        <div class="card-content">
          <h3>Uptime</h3>
          <div class="metric-value">{{ metrics.uptime }}</div>
          <div class="metric-detail">{{ formatSeconds(metrics.uptime_seconds) }}</div>
        </div>
      </div>

      <!-- Memory Usage Card -->
      <div class="metric-card">
        <div class="card-icon">💾</div>
        <div class="card-content">
          <h3>Memory Usage</h3>
          <div class="metric-value">{{ metrics.memory.usage_percent.toFixed(1) }}%</div>
          <div class="metric-detail">{{ metrics.memory.alloc_human }} / {{ metrics.memory.sys_human }}</div>
          <div class="progress-bar">
            <div class="progress-fill" :style="{width: metrics.memory.usage_percent + '%'}"></div>
          </div>
        </div>
      </div>

      <!-- Database Card -->
      <div class="metric-card">
        <div class="card-icon">🗄️</div>
        <div class="card-content">
          <h3>Database</h3>
          <div class="metric-value">{{ metrics.database.size_human }}</div>
          <div class="metric-detail">{{ metrics.database.user_count }} users, {{ metrics.database.audit_log_count }} audit logs</div>
        </div>
      </div>

      <!-- Goroutines Card -->
      <div class="metric-card">
        <div class="card-icon">🔄</div>
        <div class="card-content">
          <h3>Goroutines</h3>
          <div class="metric-value">{{ metrics.goroutines }}</div>
          <div class="metric-detail">{{ metrics.cpu.num_cpu }} CPUs available</div>
        </div>
      </div>
    </div>

    <!-- Health Checks Detail -->
    <div v-if="health" class="health-checks">
      <h2>Health Checks</h2>
      <div class="checks-grid">
        <div v-for="(check, name) in health.checks" :key="name" class="check-item" :class="check.status">
          <div class="check-icon">
            <span v-if="check.status === 'pass'">✓</span>
            <span v-else-if="check.status === 'warn'">!</span>
            <span v-else>✗</span>
          </div>
          <div class="check-details">
            <h4>{{ formatCheckName(name) }}</h4>
            <p v-if="check.message">{{ check.message }}</p>
            <p v-else class="status-text">{{ check.status }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Memory Details -->
    <div v-if="metrics" class="details-section">
      <h2>Memory Details</h2>
      <div class="details-grid">
        <div class="detail-item">
          <span class="label">Allocated:</span>
          <span class="value">{{ metrics.memory.alloc_human }}</span>
        </div>
        <div class="detail-item">
          <span class="label">Total Allocated:</span>
          <span class="value">{{ formatBytes(metrics.memory.total_alloc) }}</span>
        </div>
        <div class="detail-item">
          <span class="label">System Memory:</span>
          <span class="value">{{ metrics.memory.sys_human }}</span>
        </div>
        <div class="detail-item">
          <span class="label">Heap Allocated:</span>
          <span class="value">{{ formatBytes(metrics.memory.heap_alloc) }}</span>
        </div>
        <div class="detail-item">
          <span class="label">Heap System:</span>
          <span class="value">{{ formatBytes(metrics.memory.heap_sys) }}</span>
        </div>
        <div class="detail-item">
          <span class="label">Heap In Use:</span>
          <span class="value">{{ formatBytes(metrics.memory.heap_inuse) }}</span>
        </div>
      </div>
    </div>

    <!-- Loading Overlay -->
    <div v-if="isLoading" class="loading-overlay">
      <div class="spinner"></div>
      <p>Loading metrics...</p>
    </div>

    <!-- Toast Notification -->
    <div v-if="toast.show" class="toast" :class="toast.type">
      {{ toast.message }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { apiRequest } from '../../utils/api';

interface SystemMetrics {
  timestamp: string;
  uptime: string;
  uptime_seconds: number;
  cpu: {
    num_cpu: number;
    num_goroutine: number;
  };
  memory: {
    alloc: number;
    total_alloc: number;
    sys: number;
    heap_alloc: number;
    heap_sys: number;
    heap_inuse: number;
    alloc_human: string;
    sys_human: string;
    usage_percent: number;
  };
  database: {
    size: number;
    size_human: string;
    table_count: number;
    user_count: number;
    audit_log_count: number;
  };
  goroutines: number;
}

interface HealthCheck {
  status: string;
  message?: string;
}

interface Health {
  status: string;
  checks: Record<string, HealthCheck>;
  metrics: SystemMetrics;
  timestamp: string;
}

const metrics = ref<SystemMetrics | null>(null);
const health = ref<Health | null>(null);
const isLoading = ref(false);
const refreshInterval = ref<number | null>(null);

const toast = ref({
  show: false,
  message: '',
  type: 'success'
});

onMounted(() => {
  loadData();
  // Auto-refresh every 5 seconds
  refreshInterval.value = window.setInterval(loadData, 5000);
});

onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value);
  }
});

async function loadData() {
  try {
    // Load health status (includes metrics)
    const healthData = await apiRequest('/api/admin/system/health');
    health.value = healthData;
    metrics.value = healthData.metrics;
  } catch (error: any) {
    console.error('Failed to load system data:', error);
    if (!metrics.value) {
      // Only show error on first load
      showToast('Failed to load system metrics: ' + error.message, 'error');
    }
  }
}

async function refreshData() {
  isLoading.value = true;
  try {
    await loadData();
    showToast('Metrics refreshed', 'success');
  } catch (error: any) {
    showToast('Failed to refresh: ' + error.message, 'error');
  } finally {
    isLoading.value = false;
  }
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleTimeString();
}

function formatSeconds(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  
  if (days > 0) {
    return `${days}d ${hours}h ${mins}m`;
  } else if (hours > 0) {
    return `${hours}h ${mins}m`;
  } else {
    return `${mins}m`;
  }
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

function formatCheckName(name: string): string {
  return name.charAt(0).toUpperCase() + name.slice(1);
}

function showToast(message: string, type: 'success' | 'error') {
  toast.value = { show: true, message, type };
  setTimeout(() => {
    toast.value.show = false;
  }, 3000);
}
</script>

<style scoped>
.system-monitor {
  padding: 24px;
  max-width: 1400px;
  margin: 0 auto;
  position: relative;
}

.header {
  margin-bottom: 24px;
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

.health-banner {
  display: flex;
  align-items: center;
  padding: 20px;
  border-radius: 12px;
  margin-bottom: 24px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.health-banner.healthy {
  background: linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%);
  border: 2px solid #10b981;
}

.health-banner.degraded {
  background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
  border: 2px solid #f59e0b;
}

.health-banner.unhealthy {
  background: linear-gradient(135deg, #fee2e2 0%, #fecaca 100%);
  border: 2px solid #ef4444;
}

.status-icon {
  font-size: 48px;
  margin-right: 20px;
}

.status-details {
  flex: 1;
}

.status-details h2 {
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 4px;
}

.status-details p {
  font-size: 14px;
  color: #666;
}

.btn-refresh {
  padding: 10px 20px;
  background: white;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-refresh:hover:not(:disabled) {
  background: #f9fafb;
  border-color: #3b82f6;
}

.btn-refresh:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 20px;
  margin-bottom: 32px;
}

.metric-card {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  display: flex;
  gap: 16px;
}

.card-icon {
  font-size: 40px;
}

.card-content {
  flex: 1;
}

.card-content h3 {
  font-size: 14px;
  font-weight: 600;
  color: #666;
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.metric-value {
  font-size: 32px;
  font-weight: 700;
  color: #1a1a1a;
  margin-bottom: 4px;
}

.metric-detail {
  font-size: 14px;
  color: #666;
  margin-bottom: 8px;
}

.progress-bar {
  width: 100%;
  height: 8px;
  background: #e5e7eb;
  border-radius: 4px;
  overflow: hidden;
  margin-top: 8px;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6 0%, #2563eb 100%);
  transition: width 0.3s ease;
}

.health-checks {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  margin-bottom: 24px;
}

.health-checks h2 {
  font-size: 20px;
  font-weight: 600;
  margin-bottom: 20px;
}

.checks-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
}

.check-item {
  display: flex;
  align-items: flex-start;
  padding: 16px;
  border-radius: 8px;
  border: 2px solid #e5e7eb;
}

.check-item.pass {
  background: #f0fdf4;
  border-color: #10b981;
}

.check-item.warn {
  background: #fffbeb;
  border-color: #f59e0b;
}

.check-item.fail {
  background: #fef2f2;
  border-color: #ef4444;
}

.check-icon {
  font-size: 24px;
  font-weight: 700;
  margin-right: 12px;
}

.check-item.pass .check-icon {
  color: #10b981;
}

.check-item.warn .check-icon {
  color: #f59e0b;
}

.check-item.fail .check-icon {
  color: #ef4444;
}

.check-details h4 {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 4px;
}

.check-details p {
  font-size: 14px;
  color: #666;
}

.status-text {
  font-weight: 600;
  text-transform: uppercase;
  font-size: 12px;
}

.details-section {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.details-section h2 {
  font-size: 20px;
  font-weight: 600;
  margin-bottom: 20px;
}

.details-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  padding: 12px;
  background: #f9fafb;
  border-radius: 6px;
}

.detail-item .label {
  font-size: 14px;
  color: #666;
  font-weight: 500;
}

.detail-item .value {
  font-size: 14px;
  color: #1a1a1a;
  font-weight: 600;
}

.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.9);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.spinner {
  width: 50px;
  height: 50px;
  border: 5px solid #e5e7eb;
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 16px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
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
  .system-monitor {
    padding: 16px;
  }
  
  .health-banner {
    flex-direction: column;
    text-align: center;
  }
  
  .status-icon {
    margin-right: 0;
    margin-bottom: 12px;
  }
  
  .metrics-grid,
  .checks-grid,
  .details-grid {
    grid-template-columns: 1fr;
  }
}
</style>
