<template>
  <div class="progress">
    <div class="bar" :style="{ width: `${percent}%` }" :class="colorClass"></div>
    <div class="label">Level {{ level }}: {{ currentXP.toLocaleString() }} / {{ requiredXP.toLocaleString() }} XP ({{ percent.toFixed(0) }}%)</div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  currentXP: { type: Number, default: 0 },
  requiredXP: { type: Number, default: 1 },
  level: { type: Number, default: 1 }
})

const percent = computed(() => {
  const req = Math.max(1, props.requiredXP)
  const pct = Math.max(0, Math.min(100, (props.currentXP / req) * 100))
  return pct
})

const colorClass = computed(() => {
  if (percent.value < 33) return 'low'
  if (percent.value < 67) return 'mid'
  return 'high'
})
</script>

<style scoped>
.progress { position: relative; background: var(--bg-tertiary); border: 1px solid var(--border-color); border-radius: 8px; height: 26px; overflow: hidden; }
.bar { height: 100%; transition: width 0.4s ease; background: linear-gradient(90deg, var(--accent-gradient-start), var(--accent-gradient-end)); }
.bar.low { background: linear-gradient(90deg, #60a5fa, #3b82f6); }
.bar.mid { background: linear-gradient(90deg, #34d399, #10b981); }
.bar.high { background: linear-gradient(90deg, #f59e0b, #d97706); }
.label { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; font-size: 0.85rem; font-weight: 600; color: var(--text-primary); text-shadow: 0 1px 2px var(--shadow); }
</style>
