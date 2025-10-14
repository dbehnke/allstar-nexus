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
                <tr v-for="lvl in levels" :key="lvl"><td>{{ lvl }}</td><td>{{ xpFor(lvl) }}</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="section">
          <h4>Renown (Prestige)</h4>
          <p>When a user reaches level 60, they enter a Renown cycle. Each Renown level requires a fixed amount of XP configured on the server.</p>
          <p><strong>Default:</strong> {{ renownXP }} seconds (10 hours)</p>
          <p>When Renown is awarded the player's level is reset to 1 and any leftover XP beyond the renown threshold is carried into the new cycle.</p>
        </section>

        <section class="section">
          <h4>Rested XP</h4>
          <p>Rested XP awards a multiplier for time spent after being inactive. The server accumulates a rested bonus which is consumed on next session(s). See server config for exact accumulation and multiplier values.</p>
        </section>

        <section class="section">
          <h4>Diminishing Returns</h4>
          <p>Diminishing returns reduce XP awarded for longer daily talk time windows. The system applies tiered multipliers based on recent activity to encourage fair play.</p>
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
const props = defineProps({ visible: Boolean, levelConfig: Object, renownXP: { type: Number, default: 36000 }, renownEnabled: { type: Boolean, default: true } })
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
