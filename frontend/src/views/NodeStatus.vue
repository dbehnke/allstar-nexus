<template>
  <div class="node-status-page">
    <StatusCard :status="nodeStore.status" @refresh="refreshStats" />
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useNodeStore } from '../stores/node'
import StatusCard from '../components/StatusCard.vue'
import { logger } from '../utils/logger'

const nodeStore = useNodeStore()

function refreshStats() {
  // Refresh happens via WebSocket, this is just for manual refresh button
  logger.info('Manual refresh requested')
}

onMounted(() => {
  nodeStore.startTickTimer()
})
</script>

<style scoped>
.node-status-page {
  padding: 1rem;
  max-width: 1600px;
  margin: 0 auto;
}
</style>
