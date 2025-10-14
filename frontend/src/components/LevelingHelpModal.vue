<template>
  <div v-if="visible" class="modal-overlay" @click.self="close">
    <div class="modal" role="dialog" aria-modal="true">
      <div class="modal-header">
        <h3>How Leveling Works</h3>
        <button class="close" @click="close">✕</button>
      </div>
      <div class="modal-body">
        <section class="section">
          <h4>Levels and XP</h4>
          <p>This table shows XP required to advance from the current level to the next level. Values are seconds of credited talk time.</p>
          <div class="table-wrap">
            <table>
              <thead><tr><th>Level</th><th>XP to next level (seconds)</th></tr></thead>
              <tbody>
                  <tr v-for="lvl in levels" :key="lvl">
                    <td>{{ lvl }}</td>
                    <td>
                      {{ xpFor(lvl) }}
                      <small class="muted">({{ formatTime(xpFor(lvl)) }})</small>
                    </td>
                  </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="section">
          <h4>Renown (Prestige)</h4>
          <p>When a user reaches level 60, they enter a Renown cycle. Each Renown level requires a fixed amount of XP configured on the server.</p>
          <p><strong>Default:</strong> {{ renownXP }} seconds (10 hours)</p>
          <p>When Renown is awarded the player's level is reset to 1 and any leftover XP beyond the renown threshold is carried into the new cycle.</p>
            <p v-if="weeklyCapSeconds != null"><strong>Weekly level cap:</strong> {{ weeklyCapSeconds }} seconds (~{{ formatTime(weeklyCapSeconds) }})</p>
        </section>

        <section class="section">
          <h4>Rested XP</h4>
          <div v-if="!restedEnabled">
            <p>Rested XP is currently disabled on this server.</p>
          </div>
          <div v-else>
            <p>Rested XP awards a multiplier for time spent after being inactive. The server accumulates a rested bonus which is consumed on next session(s).</p>
            <ul>
              <li><strong>Accumulation rate:</strong> {{ restedAccumulationRate }} hours bonus per hour idle</li>
              <li><strong>Maximum cap:</strong> {{ restedMaxHours }} hours</li>
              <li><strong>Multiplier when rested:</strong> {{ formatMultiplier(restedMultiplier) }}</li>
            </ul>
          </div>
        </section>

        <section class="section">
          <h4>Diminishing Returns</h4>
          <p>Diminishing returns reduce XP awarded for longer daily talk time windows. The system applies tiered multipliers based on recent activity to encourage fair play.</p>
            <p>Example tiers (server-configurable):</p>
            <ul>
              <li>0–20 minutes: 1.0x (full XP)</li>
              <li>20–40 minutes: 0.75x</li>
              <li>40–60 minutes: 0.5x</li>
              <li>60+ minutes: 0.25x</li>
            </ul>
            <p>This means longer, continuous talk sessions earn less XP per second after passing each tier threshold. The intent is to reward diversified participation while limiting farming of long continuous TX time.</p>
        </section>
      </div>
      <div class="modal-footer">
        <button class="btn-primary" @click="close">Close</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
const props = defineProps({
  visible: Boolean,
  levelConfig: Object,
  renownXP: { type: Number, default: 36000 },
  renownEnabled: { type: Boolean, default: true },
  weeklyCapSeconds: { type: Number, default: null },
  // Rested server values (provided by backend API)
  restedEnabled: { type: Boolean, default: false },
  restedAccumulationRate: { type: Number, default: 0 },
  restedMaxHours: { type: Number, default: 0 },
  restedMultiplier: { type: Number, default: 1.0 },
})
const emits = defineEmits(['close'])

function close() { emits('close') }

const levels = computed(() => {
  const lc = props.levelConfig || {}
  // show 1..60
  const arr = []
  for (let i = 1; i <= 60; i++) arr.push(i)
  return arr
})

function xpFor(lvl) {
  const lc = props.levelConfig || {}
  const k = String(lvl)
  if (lc[k] != null) return lc[k]
  if (lvl <= 10) return 360
  return 360 + Math.floor(Math.pow(lvl - 10, 1.8) * 100)
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
  const h = (s / 3600)
  return `${h.toFixed(1)} hour${h.toFixed(1) === '1.0' ? '' : 's'}`
}

function formatMultiplier(n) {
  const v = Number(n)
  if (isNaN(v)) return '-'
  return `${v.toFixed(1)}x`
}
</script>

<style scoped>
.modal-overlay { position: fixed; inset: 0; background: rgba(15,15,15,0.5); display:flex; align-items:center; justify-content:center; z-index:50 }
.modal { width: min(900px, 95%); max-height: 80vh; overflow: auto; background: var(--bg-secondary); border-radius: 8px; padding: 1rem; box-shadow: 0 10px 30px rgba(0,0,0,0.4); }
.modal-header { display:flex; align-items:center; justify-content:space-between; }
.modal-body { padding: 0.5rem 0; }
.modal-footer { display:flex; justify-content:flex-end; margin-top: 0.5rem }
.close { background: transparent; border: 0; font-size: 1.1rem }
.table-wrap { max-height: 36vh; overflow: auto; border: 1px solid var(--border-color); border-radius: 6px }
table { width: 100%; border-collapse: collapse }
th, td { padding: 0.35rem 0.5rem; text-align: left; border-bottom: 1px solid var(--border-color) }
.section { margin-bottom: 1rem }
</style>
