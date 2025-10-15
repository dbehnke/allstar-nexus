<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="close">
      <div class="modal" role="dialog" aria-modal="true">
        <div class="modal-header">
          <h3>{{ callsign }}</h3>
          <button class="close" @click="close">✕</button>
        </div>
        <div class="modal-body">
          <div v-if="loading" class="loading">Loading profile...</div>
          <div v-else-if="error" class="error">{{ error }}</div>
          <div v-else-if="!profile" class="loading">No profile data available</div>
          <div v-else>
            <section class="section">
              <h4>Level Progress</h4>
              <div class="stat-row">
                <span class="label">Level:</span>
                <span class="value">{{ profile.level || 1 }}</span>
              </div>
              <div v-if="profile.renown_level > 0" class="stat-row">
                <span class="label">Renown:</span>
                <span class="value renown">⭐ {{ profile.renown_level }}</span>
              </div>
              <div class="stat-row">
                <span class="label">Current XP:</span>
                <span class="value">{{ profile.experience_points || 0 }} / {{ profile.next_level_xp || 0 }}</span>
              </div>
              <div class="stat-row">
                <span class="label">Total Talk Time:</span>
                <span class="value">{{ formatTime(profile.total_talk_time_seconds || 0) }}</span>
              </div>
            </section>

            <section class="section">
              <h4>Rested XP</h4>
              <div class="stat-row">
                <span class="label">Available:</span>
                <span class="value" :class="{ 'highlight': (profile.rested_bonus_seconds || 0) > 0 }">
                  {{ formatTime(profile.rested_bonus_seconds || 0) }}
                </span>
              </div>
              <div v-if="(profile.rested_bonus_seconds || 0) > 0" class="info-text">
                Your next {{ formatTime(profile.rested_bonus_seconds) }} of talk time will earn bonus XP!
              </div>
              <div v-else class="info-text muted">
                No rested XP available. Rested XP accumulates while you're idle.
              </div>
            </section>

            <section class="section">
              <h4>XP Caps</h4>
              <div class="stat-row">
                <span class="label">Daily XP:</span>
                <span class="value" :class="capStatusClass('daily')">
                  {{ profile.daily_xp || 0 }} / {{ dailyCapSeconds || 0 }} seconds
                  <span v-if="isDailyCapped" class="badge capped">CAPPED</span>
                </span>
              </div>
              <div class="stat-row">
                <span class="label">Weekly XP:</span>
                <span class="value" :class="capStatusClass('weekly')">
                  {{ profile.weekly_xp || 0 }} / {{ weeklyCapSeconds || 0 }} seconds
                  <span v-if="isWeeklyCapped" class="badge capped">CAPPED</span>
                </span>
              </div>
              <div v-if="isDailyCapped || isWeeklyCapped" class="warning-text">
                You've reached your {{ isDailyCapped ? 'daily' : 'weekly' }} XP cap. Further transmissions won't earn XP until the cap resets.
              </div>
            </section>

            <section v-if="drTiers.length > 0" class="section">
              <h4>Diminishing Returns</h4>
              <div class="info-text">
                Based on your recent activity, XP is currently multiplied by: <strong>{{ currentDRMultiplier }}x</strong>
              </div>
              <div class="dr-tiers">
                <div v-for="(tier, idx) in drTiers" :key="idx" class="tier" :class="{ 'active': isActiveTier(tier) }">
                  <span class="tier-range">{{ formatTierRange(tier, idx) }}</span>
                  <span class="tier-mult">{{ tier.multiplier }}x</span>
                </div>
              </div>
            </section>

            <section class="section">
              <h4>Recent Activity</h4>
              <div v-if="profile.last_transmission_at" class="stat-row">
                <span class="label">Last Active:</span>
                <span class="value">{{ formatTimestamp(profile.last_transmission_at) }}</span>
              </div>
            </section>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-primary" @click="close">Close</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  visible: Boolean,
  callsign: String,
  dailyCapSeconds: { type: Number, default: 1200 },
  weeklyCapSeconds: { type: Number, default: 7200 },
  drTiers: { type: Array, default: () => [] }
})

const emits = defineEmits(['close'])

const loading = ref(false)
const error = ref(null)
const profile = ref(null)

watch(() => props.visible, (newVal) => {
  if (newVal && props.callsign) {
    profile.value = null
    error.value = null
    fetchProfile()
  } else if (!newVal) {
    // Reset state when modal closes
    profile.value = null
    error.value = null
    loading.value = false
  }
})

async function fetchProfile() {
  if (!props.callsign) return
  loading.value = true
  error.value = null
  try {
    const endpoint = `/api/gamification/profile/${encodeURIComponent(props.callsign)}`
    const base = (typeof globalThis !== 'undefined' && globalThis.location && globalThis.location.origin) ? globalThis.location.origin : 'http://localhost'
    const res = await fetch(new URL(endpoint, base).toString())
    if (!res.ok) throw new Error('Failed to fetch profile')
    const data = await res.json()
    profile.value = data
  } catch (e) {
    error.value = 'Failed to load profile details'
    console.error('Failed to fetch profile', props.callsign, e)
  } finally {
    loading.value = false
  }
}

function close() {
  emits('close')
}

const isDailyCapped = computed(() => {
  if (!profile.value) return false
  return (profile.value.daily_xp || 0) >= props.dailyCapSeconds
})

const isWeeklyCapped = computed(() => {
  if (!profile.value) return false
  return (profile.value.weekly_xp || 0) >= props.weeklyCapSeconds
})

function capStatusClass(type) {
  if (type === 'daily' && isDailyCapped.value) return 'capped-text'
  if (type === 'weekly' && isWeeklyCapped.value) return 'capped-text'
  return ''
}

const currentDRMultiplier = computed(() => {
  if (!profile.value || !props.drTiers || props.drTiers.length === 0) return '1.0'
  const dailySeconds = profile.value.daily_xp || 0
  for (const tier of props.drTiers) {
    if (tier && tier.max_seconds != null && dailySeconds <= tier.max_seconds) {
      return (tier.multiplier != null ? tier.multiplier : 1.0).toFixed(2)
    }
  }
  const lastTier = props.drTiers[props.drTiers.length - 1]
  return (lastTier && lastTier.multiplier != null ? lastTier.multiplier : 1.0).toFixed(2)
})

function isActiveTier(tier) {
  if (!profile.value || !tier || tier.max_seconds == null) return false
  const dailySeconds = profile.value.daily_xp || 0
  return dailySeconds <= tier.max_seconds
}

function formatTierRange(tier, idx) {
  if (!tier || tier.max_seconds == null) return '-'
  if (idx === 0) {
    return `0–${formatTime(tier.max_seconds)}`
  }
  const prevTier = props.drTiers[idx - 1]
  if (!prevTier || prevTier.max_seconds == null) return formatTime(tier.max_seconds)
  const prevMax = prevTier.max_seconds
  return `${formatTime(prevMax)}–${formatTime(tier.max_seconds)}`
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

function formatTimestamp(ts) {
  if (!ts) return '-'
  try {
    const date = new Date(ts)
    const now = new Date()
    const diff = now - date
    const seconds = Math.floor(diff / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)

    if (days > 0) return `${days} day${days === 1 ? '' : 's'} ago`
    if (hours > 0) return `${hours} hour${hours === 1 ? '' : 's'} ago`
    if (minutes > 0) return `${minutes} minute${minutes === 1 ? '' : 's'} ago`
    return 'Just now'
  } catch (e) {
    return ts
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(15, 15, 15, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}

.modal {
  width: min(600px, 95%);
  max-height: 80vh;
  overflow: auto;
  background: var(--bg-secondary);
  border-radius: 8px;
  padding: 1rem;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.modal-body {
  padding: 0.5rem 0;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}

.close {
  background: transparent;
  border: 0;
  font-size: 1.1rem;
  cursor: pointer;
  color: var(--text-primary);
}

.section {
  margin-bottom: 1.5rem;
}

.section h4 {
  margin-bottom: 0.5rem;
  color: var(--accent-primary);
  font-size: 1.1rem;
}

.stat-row {
  display: flex;
  justify-content: space-between;
  padding: 0.5rem 0;
  border-bottom: 1px solid var(--border-color);
}

.stat-row .label {
  font-weight: 600;
  color: var(--text-secondary);
}

.stat-row .value {
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.value.highlight {
  color: #10b981;
  font-weight: 700;
}

.value.renown {
  color: #f59e0b;
  font-weight: 700;
}

.capped-text {
  color: #ef4444 !important;
  font-weight: 700;
}

.badge {
  font-size: 0.7rem;
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
  font-weight: 700;
}

.badge.capped {
  background: #dc2626;
  color: white;
}

.info-text {
  margin-top: 0.5rem;
  padding: 0.5rem;
  background: var(--bg-tertiary);
  border-radius: 4px;
  font-size: 0.9rem;
  color: var(--text-secondary);
}

.warning-text {
  margin-top: 0.5rem;
  padding: 0.5rem;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid #ef4444;
  border-radius: 4px;
  font-size: 0.9rem;
  color: #ef4444;
}

.muted {
  color: var(--text-muted);
}

.loading,
.error {
  padding: 2rem;
  text-align: center;
}

.error {
  color: #ef4444;
}

.dr-tiers {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.tier {
  display: flex;
  justify-content: space-between;
  padding: 0.5rem;
  background: var(--bg-tertiary);
  border-radius: 4px;
  border: 1px solid var(--border-color);
}

.tier.active {
  border-color: var(--accent-primary);
  background: rgba(56, 189, 248, 0.1);
}

.tier-range {
  color: var(--text-secondary);
}

.tier-mult {
  font-weight: 700;
  color: var(--text-primary);
}
</style>
