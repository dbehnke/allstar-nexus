<template>
  <div class="node-status-page">
    <StatusCard :status="nodeStore.status" @refresh="refreshStats" />
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useNodeStore } from '../stores/node'
import StatusCard from '../components/StatusCard.vue'

const nodeStore = useNodeStore()

function refreshStats() {
  // Refresh happens via WebSocket, this is just for manual refresh button
  console.log('Manual refresh requested')
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
