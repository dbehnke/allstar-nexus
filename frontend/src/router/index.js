import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
// Removed NodeStatus route; replaced with Talker Log
import NodeLookup from '../views/NodeLookup.vue'
import RptStats from '../views/RptStats.vue'
import VoterDisplay from '../views/VoterDisplay.vue'
import NetworkMap from '../views/NetworkMap.vue'
import AdminDashboard from '../views/Admin/AdminDashboard.vue'
import ConfigEditor from '../views/Admin/ConfigEditor.vue'
import CommandPanel from '../views/Admin/CommandPanel.vue'
import NodeControl from '../views/Admin/NodeControl.vue'
import UserManager from '../views/Admin/UserManager.vue'
import DatabaseBackup from '../views/Admin/DatabaseBackup.vue'
import SystemMonitor from '../views/Admin/SystemMonitor.vue'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/',
    name: 'Dashboard',
    component: Dashboard
  },
  {
    path: '/lookup',
    name: 'NodeLookup',
    component: NodeLookup
  },
  {
    path: '/network-map',
    name: 'NetworkMap',
    component: NetworkMap
  },
  {
    path: '/rpt-stats',
    name: 'RptStats',
    component: RptStats
  },
  {
    path: '/voter',
    name: 'Voter',
    component: VoterDisplay
  },
  {
    path: '/admin',
    name: 'Admin',
    component: AdminDashboard,
    meta: { requiresSuperAdmin: true }
  },
  {
    path: '/admin/config',
    name: 'ConfigEditor',
    component: ConfigEditor,
    meta: { requiresSuperAdmin: true }
  },
  {
    path: '/admin/commands',
    name: 'CommandPanel',
    component: CommandPanel,
    meta: { requiresSuperAdmin: true }
  },
  {
    path: '/admin/nodes',
    name: 'NodeControl',
    component: NodeControl,
    meta: { requiresSuperAdmin: true }
  },
  {
    path: '/admin/users',
    name: 'UserManager',
    component: UserManager,
    meta: { requiresSuperAdmin: true }
  },
  {
    path: '/admin/database',
    name: 'DatabaseBackup',
    component: DatabaseBackup,
    meta: { requiresSuperAdmin: true }
  },
  {
    path: '/admin/system',
    name: 'SystemMonitor',
    component: SystemMonitor,
    meta: { requiresSuperAdmin: true }
  }
]

const router = createRouter({
  history: createWebHistory('/'),
  routes
})

// Navigation guard for admin routes
router.beforeEach((to, from, next) => {
  if (to.meta.requiresSuperAdmin) {
    const authStore = useAuthStore()
    if (!authStore.isAuthenticated || authStore.role !== 'superadmin') {
      // Redirect to dashboard if not superadmin
      next('/')
      return
    }
  }
  next()
})

export default router
