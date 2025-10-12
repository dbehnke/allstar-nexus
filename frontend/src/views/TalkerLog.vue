<template>
  <div class="talker-log-page">
    <h2 class="page-title">Talker Log</h2>
    <div class="grid-layout">
      <TransmissionHistoryCard
        :transmissions="recentTransmissions"
        :currentPage="currentPage"
        :totalPages="totalPages"
        @page-change="handlePageChange"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import TransmissionHistoryCard from '../components/TransmissionHistoryCard.vue'
import { useNodeStore } from '../stores/node'

const nodeStore = useNodeStore()
const scoreboard = nodeStore.scoreboard
const recentTransmissions = nodeStore.recentTransmissions
const levelConfig = nodeStore.levelConfig
const currentPage = ref(1)
const totalPages = ref(5)

function handlePageChange(page) {
  currentPage.value = page
  // Our API uses limit 50/offset; keep same page calc
  nodeStore.fetchRecentTransmissions(50, (page - 1) * 50)
}

function refreshScoreboard() {
  nodeStore.fetchScoreboard(50)
}

let intervalId = null
onMounted(() => {
  // Scoreboard is displayed on the Dashboard; still fetch recent transmissions here
  nodeStore.fetchRecentTransmissions(50, 0)
  nodeStore.fetchLevelConfig()
  // periodic refresh every 60s to keep list fresh even if WS missed
  intervalId = setInterval(() => {
    nodeStore.fetchRecentTransmissions(50, (currentPage.value - 1) * 50)
  }, 60000)
})

onUnmounted(() => {
  if (intervalId) clearInterval(intervalId)
})
</script>

<style scoped>
.talker-log-page {
  padding: 1rem;
  max-width: 1600px;
  margin: 0 auto;
}

.page-title { margin: 0 0 0.5rem 0; font-size: 1.25rem; color: var(--text-secondary); }

.grid-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

@media (max-width: 1024px) {
  .grid-layout {
    grid-template-columns: 1fr;
  }
}
</style>
