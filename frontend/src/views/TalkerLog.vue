<template>
  <div class="talker-log-page">
    <div class="grid-layout">
      <TransmissionHistoryCard
        :transmissions="recentTransmissions"
        :currentPage="currentPage"
        :totalPages="totalPages"
        @page-change="handlePageChange"
      />

      <ScoreboardCard
        :scoreboard="scoreboard"
        :level-config="levelConfig"
        @refresh="refreshScoreboard"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import TransmissionHistoryCard from '../components/TransmissionHistoryCard.vue'
import ScoreboardCard from '../components/ScoreboardCard.vue'
import { useAuthStore } from '../stores/auth'

const scoreboard = ref([])
const recentTransmissions = ref([])
const currentPage = ref(1)
const totalPages = ref(5)
const levelConfig = ref({})

async function fetchScoreboard() {
  const auth = useAuthStore()
  const res = await fetch('/api/gamification/scoreboard?limit=50', {
    headers: auth.getAuthHeaders()
  })
  const data = await res.json()
  scoreboard.value = (data && (data.scoreboard || data.data || data.results)) || []
}

async function fetchRecentTransmissions(page = 1) {
  const limit = 10
  const offset = (page - 1) * limit
  const auth = useAuthStore()
  const res = await fetch(`/api/gamification/recent-transmissions?limit=50&offset=${offset}`, {
    headers: auth.getAuthHeaders()
  })
  const data = await res.json()
  recentTransmissions.value = (data && (data.transmissions || data.data || data.results)) || []
  totalPages.value = Math.ceil(50 / limit)
}

async function fetchLevelConfig() {
  try {
    const res = await fetch('/api/gamification/level-config')
    const data = await res.json()
    levelConfig.value = (data && (data.config || data.data || {})) || {}
  } catch {}
}

function handlePageChange(page) {
  currentPage.value = page
  fetchRecentTransmissions(page)
}

function refreshScoreboard() {
  fetchScoreboard()
}

onMounted(() => {
  fetchScoreboard()
  fetchRecentTransmissions(1)
  fetchLevelConfig()
})
</script>

<style scoped>
.talker-log-page {
  padding: 1rem;
  max-width: 1600px;
  margin: 0 auto;
}

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
