<template>
  <div class="card">
    <div class="card-header">
      <div class="header-left">
        <h3>Achievements</h3>
      </div>
        <div class="header-right">
          <div v-if="props.renownEnabled" class="renown-indicator" title="Renown enabled on this server">
            ⭐ Renown: {{ Math.round(props.renownXP/3600) }}h
          </div>
          <button class="btn-secondary" @click="$emit('refresh')">Refresh</button>
          <button class="btn-secondary" @click="showHelp = true" title="How leveling works">?</button>
        </div>
    </div>
    <div class="card-body">
      <div v-if="(scoreboard || []).length" class="scoreboard-list">
        <div v-for="(p, i) in scoreboard" :key="p.callsign || i" class="entry" :class="rankClass(i)">
          <div class="badge-container">
            <div v-if="(p.renown_level || 0) > 0" class="group-badge renown-badge">⭐</div>
            <div v-else-if="p.grouping" class="group-badge" :style="{ borderColor: p.grouping.color || '#64748b' }">
              {{ p.grouping.badge || '📻' }}
            </div>
            <div v-else class="group-badge">{{ i + 1 }}</div>
          </div>
          <div class="info">
            <div class="line-1">
              <a v-if="p.callsign" class="callsign" :href="`https://www.qrz.com/db/${(p.callsign||'').toUpperCase()}`" target="_blank" rel="noopener noreferrer">{{ p.callsign }}</a>
              <span v-else class="callsign">Unknown</span>
              <span v-if="(p.renown_level || 0) > 0" class="level-badge renown">⭐ Renown {{ p.renown_level }}</span>
              <span v-else-if="p.grouping" class="level-badge" :style="{ borderColor: p.grouping.color || '#64748b', color: p.grouping.color || '#64748b' }">
                {{ p.grouping.title }} • Level {{ p.level || 1 }}
              </span>
              <span v-else class="level-badge">Level {{ p.level || 1 }}</span>
            </div>
            <div class="line-2">
              <LevelProgressBar
                :current-xp="p.experience_points || p.xp || 0"
                :required-xp="requiredXP(p.level || 1)"
                :level="p.level || 1"
              />
              <div class="entry-meta">
                <button class="btn-link" @click.prevent="fetchProfile(p.callsign)">Details</button>
                <span v-if="profileDetails[p.callsign] && profileDetails[p.callsign].rested_bonus_seconds != null" class="rested">
                  Rested: {{ formatTime(profileDetails[p.callsign].rested_bonus_seconds) }}
                </span>
                <span v-else-if="profileLoading[p.callsign]" class="rested">Loading...</span>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="empty">No profiles yet</div>
    </div>
  </div>
  <LevelingHelpModal
    :visible="showHelp"
  :levelConfig="levelConfig"
  :renownXP="renownXP"
  :renownEnabled="renownEnabled"
  :weeklyCapSeconds="nodeStore.weeklyCapSeconds"
  :restedEnabled="restedEnabled"
  :restedAccumulationRate="restedAccumulationRate"
  :restedMaxHours="restedMaxHours"
  :restedMultiplier="restedMultiplier"
  :restedIdleThresholdSeconds="nodeStore.restedIdleThresholdSeconds"
    @close="showHelp = false"
  />
</template>

<script setup>
import { ref, computed } from 'vue'
import { useNodeStore } from '../stores/node'
import LevelProgressBar from './LevelProgressBar.vue'
import LevelingHelpModal from './LevelingHelpModal.vue'

const props = defineProps({
  scoreboard: { type: Array, default: () => [] },
  levelConfig: { type: Object, default: () => ({}) },
  renownXP: { type: Number, default: 36000 },
  renownEnabled: { type: Boolean, default: false }
})

const showHelpRef = ref(false)
const showHelp = computed({
  get: () => showHelpRef.value,
  set: (v) => (showHelpRef.value = v)
})

// Local cache for fetched profile details (callsign -> profile payload)
const profileDetails = ref({})
const profileLoading = ref({})

// Read server-provided rested/renown metadata from the node store so the modal
// displays the authoritative values (fixes issue where modal showed "rested disabled"
// because the parent wasn't passing the store values).
const nodeStore = useNodeStore()
const restedEnabled = computed(() => nodeStore.restedEnabled)
const restedAccumulationRate = computed(() => nodeStore.restedAccumulationRate)
const restedMaxHours = computed(() => nodeStore.restedMaxHours)
const restedMultiplier = computed(() => nodeStore.restedMultiplier)

async function fetchProfile(callsign) {
  if (!callsign) return
  if (profileDetails.value[callsign] || profileLoading.value[callsign]) return
  profileLoading.value[callsign] = true
  try {
    const endpoint = `/api/gamification/profile/${encodeURIComponent(callsign)}`
    const base = (typeof globalThis !== 'undefined' && globalThis.location && globalThis.location.origin) ? globalThis.location.origin : 'http://localhost'
    const res = await fetch(new URL(endpoint, base).toString())
    if (!res.ok) throw new Error('fetch failed')
    const data = await res.json()
    profileDetails.value[callsign] = data
  } catch (e) {
    // fail silently; don't block UI
    console.debug('failed to fetch profile', callsign, e)
  } finally {
    profileLoading.value[callsign] = false
  }
}

function formatTime(seconds) {
  if (seconds == null) return '-'
  const s = Number(seconds)
  if (isNaN(s)) return '-'
  if (s < 60) return `${s} second${s === 1 ? '' : 's'}`
  if (s < 3600) {
    const m = Math.round(s / 60)
    return `${m} minute${m === 1 ? '' : 's'}`
  }
  const h = s / 3600
  return `${h.toFixed(1)} hour${h.toFixed(1) === '1.0' ? '' : 's'}`
}

function requiredXP(level) {
  // levelConfig may be a map of string keys -> int
  const lc = props.levelConfig || {}
  const k = String(level)
  if (lc[k] != null) return lc[k]
  // fallback: simple scaling
  if (level <= 10) return 360
  return 360 + Math.floor(Math.pow(level - 10, 1.8) * 100)
}

function rankClass(index) {
  if (index === 0) return 'gold'
  if (index === 1) return 'silver'
  if (index === 2) return 'bronze'
  return ''
}
</script>

<style scoped>
.card { background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; box-shadow: 0 2px 8px var(--shadow); display: flex; flex-direction: column; }
.card-header { padding: 0.75rem 1rem; border-bottom: 1px solid var(--border-color); display: flex; align-items: center; justify-content: space-between; }
.card-body { padding: 0.75rem 1rem; }

.scoreboard-list { display: grid; grid-template-columns: repeat(2, 1fr); gap: 0.5rem; max-height: 540px; overflow: auto; }
.entry { display: grid; grid-template-columns: 56px 1fr; gap: 0.75rem; align-items: center; padding: 0.5rem; border: 1px solid var(--border-color); border-radius: 8px; background: var(--bg-tertiary); }

/* Mobile: single column */
@media (max-width: 1024px) {
  .scoreboard-list {
    grid-template-columns: 1fr;
  }
}

/* Badge container and styles */
.badge-container { width: 56px; display: flex; justify-content: center; }
.group-badge { width: 44px; height: 44px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-weight: 800; font-size: 1.6rem; background: transparent; border: 3px solid var(--border-color); box-shadow: 0 2px 8px var(--shadow); }
.group-badge.renown-badge { border-color: #f59e0b; font-size: 1.7rem; box-shadow: 0 2px 8px rgba(245, 158, 11, 0.3); }

/* Top 3 rank styling (gold/silver/bronze borders) */
.entry.gold { border-color: #f59e0b; border-width: 2px; }
.entry.silver { border-color: #9ca3af; border-width: 2px; }
.entry.bronze { border-color: #c2410c; border-width: 2px; }

.info { display: flex; flex-direction: column; gap: 0.35rem; }
.line-1 { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.callsign { color: var(--accent-primary); text-decoration: none; font-weight: 700; }
.callsign:hover { text-decoration: underline; }
.level-badge { background: var(--bg-hover); border: 1px solid var(--border-color); border-radius: 999px; padding: 0.1rem 0.5rem; font-size: 0.8rem; color: var(--text-secondary); white-space: nowrap; }
.level-badge.renown { background: rgba(245, 158, 11, 0.1); border-color: #f59e0b; color: #f59e0b; font-weight: 700; }

.empty { padding: 1rem; color: var(--text-muted); }
</style>
