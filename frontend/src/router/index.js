import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import NodeStatus from '../views/NodeStatus.vue'
import NodeLookup from '../views/NodeLookup.vue'
import RptStats from '../views/RptStats.vue'
import VoterDisplay from '../views/VoterDisplay.vue'

const routes = [
  {
    path: '/',
    name: 'Dashboard',
    component: Dashboard
  },
  {
    path: '/status',
    name: 'NodeStatus',
    component: NodeStatus
  },
  {
    path: '/lookup',
    name: 'NodeLookup',
    component: NodeLookup
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
  }
]

const router = createRouter({
  history: createWebHistory('/'),
  routes
})

export default router
