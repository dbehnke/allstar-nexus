<template>
  <div class="card">
    <div class="card-header">
      <div class="header-left">
        <h3>Scoreboard</h3>
      </div>
      <div class="header-right">
        <button class="btn-secondary" @click="$emit('refresh')">Refresh</button>
      </div>
    </div>
    <div class="card-body">
      <div v-if="(scoreboard || []).length" class="scoreboard-list">
        <div v-for="(p, i) in scoreboard" :key="p.callsign || i" class="entry" :class="rankClass(i)">
          <div class="rank-badge">{{ i + 1 }}</div>
          <div class="info">
            <div class="line-1">
              <a v-if="p.callsign" class="callsign" :href="`https://www.qrz.com/db/${(p.callsign||'').toUpperCase()}`" target="_blank" rel="noopener noreferrer">{{ p.callsign }}</a>
              <span v-else class="callsign">Unknown</span>
              <span class="level">Level {{ p.level || 1 }}</span>
              <span v-if="(p.renown_level || p.renown || 0) > 0" class="renown">⭐ Renown {{ p.renown_level || p.renown }}</span>
            </div>
            <div class="line-2">
              <LevelProgressBar
                :current-x-p="p.experience_points || p.xp || 0"
                :required-x-p="requiredXP(p.level || 1)"
                :level="p.level || 1"
              />
            </div>
          </div>
        </div>
      </div>
      <div v-else class="empty">No profiles yet</div>
    </div>
  </div>
</template>

<script setup>
import LevelProgressBar from './LevelProgressBar.vue'

const props = defineProps({
  scoreboard: { type: Array, default: () => [] },
  levelConfig: { type: Object, default: () => ({}) }
})

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

.scoreboard-list { display: flex; flex-direction: column; gap: 0.5rem; max-height: 540px; overflow: auto; }
.entry { display: grid; grid-template-columns: 56px 1fr; gap: 0.75rem; align-items: center; padding: 0.5rem; border: 1px solid var(--border-color); border-radius: 8px; background: var(--bg-tertiary); }
.rank-badge { width: 44px; height: 44px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-weight: 800; font-size: 1.1rem; color: #fff; background: linear-gradient(135deg, #64748b, #334155); box-shadow: 0 2px 8px var(--shadow); }
.entry.gold .rank-badge { background: linear-gradient(135deg, #f59e0b, #b45309); }
.entry.silver .rank-badge { background: linear-gradient(135deg, #9ca3af, #6b7280); }
.entry.bronze .rank-badge { background: linear-gradient(135deg, #c2410c, #7c2d12); }

.info { display: flex; flex-direction: column; gap: 0.35rem; }
.line-1 { display: flex; align-items: center; gap: 0.5rem; }
.callsign { color: var(--accent-primary); text-decoration: none; font-weight: 700; }
.callsign:hover { text-decoration: underline; }
.level { background: var(--bg-hover); border: 1px solid var(--border-color); border-radius: 999px; padding: 0.1rem 0.5rem; font-size: 0.8rem; color: var(--text-secondary); }
.renown { color: #f59e0b; font-weight: 700; }

.empty { padding: 1rem; color: var(--text-muted); }
</style>
