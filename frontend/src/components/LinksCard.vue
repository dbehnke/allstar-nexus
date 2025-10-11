<template>
  <Card>
    <template #header>
      <div class="card-header-content">
        <h3>{{ cardTitle }}</h3>
        <div class="header-right">
          <span v-if="anyLinkTransmitting" class="tx-indicator">
            <span class="tx-pulse"></span>
            TX ACTIVE
          </span>
          <button @click="showNotificationSettings = !showNotificationSettings" class="settings-btn" title="Notification Settings">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"></path>
              <circle cx="12" cy="12" r="3"></circle>
            </svg>
          </button>
        </div>
      </div>
      
      <!-- Notification Settings Panel -->
      <div v-if="showNotificationSettings" class="notification-settings">
        <div class="setting-row">
          <label class="setting-label">
            <input type="checkbox" v-model="notificationsEnabled" @change="onNotificationToggle" />
            <span>Enable TX Notifications</span>
          </label>
        </div>
        <div v-if="notificationsEnabled" class="setting-row">
          <label class="setting-label">
            <span>Cooldown Period:</span>
            <select v-model="notificationCooldown" class="cooldown-select">
              <option :value="30">30 seconds</option>
              <option :value="60">1 minute</option>
              <option :value="120">2 minutes</option>
              <option :value="300">5 minutes</option>
              <option :value="600">10 minutes</option>
            </select>
          </label>
          <p class="setting-help">System must be idle (no TX) for this long before you'll be notified of new activity</p>
        </div>
        <div v-if="notificationsEnabled" class="setting-row">
          <label class="setting-label">
            <input type="checkbox" v-model="soundEnabled" @change="onSoundToggle" />
            <span>Play notification sound</span>
          </label>
        </div>
        <div v-if="notificationsEnabled && soundEnabled" class="setting-row">
          <label class="setting-label">
            <span>Sound Type:</span>
            <select v-model="soundType" @change="onSoundTypeChange" class="sound-select">
              <option value="two-tone">Two-Tone Beep</option>
              <option value="single-beep">Single Beep</option>
              <option value="triple-beep">Triple Beep</option>
              <option value="ascending">Ascending Tones</option>
              <option value="descending">Descending Tones</option>
              <option value="chirp">Chirp</option>
            </select>
          </label>
        </div>
        <div v-if="notificationsEnabled && soundEnabled" class="setting-row">
          <label class="setting-label volume-control">
            <span>🔊 Volume:</span>
            <input 
              type="range" 
              v-model="soundVolume" 
              @input="onVolumeChange"
              min="0" 
              max="100" 
              class="volume-slider"
            />
            <span class="volume-value">{{ soundVolume }}%</span>
          </label>
        </div>
        <div v-if="notificationsEnabled" class="setting-row">
          <label class="setting-label">
            <input type="checkbox" v-model="speechEnabled" @change="onSpeechToggle" />
            <span>Use speech synthesis</span>
          </label>
        </div>
        <div v-if="notificationsEnabled" class="setting-row button-row">
          <button @click="sendTestNotification" class="test-notification-btn">🔔 Test Notification</button>
          <button v-if="soundEnabled" @click="playNotificationSound" class="test-notification-btn secondary">🔊 Test Sound</button>
          <button v-if="speechEnabled" @click="testSpeech" class="test-notification-btn secondary">🗣️ Test Speech</button>
        </div>
        <div v-if="notificationsEnabled && cooldownRemaining > 0" class="cooldown-status">
          ⏱️ Cooldown active: {{ formatCooldownTime(cooldownRemaining) }} remaining
        </div>
        <div v-if="notificationsEnabled" class="debug-info">
          <small style="color: var(--text-muted);">
            Debug: Permission={{ notificationPermission }}, 
            LastNotif={{ lastNotificationTime }}, 
            Cooldown={{ cooldownRemaining }}s
          </small>
        </div>
        <div v-if="notificationPermission === 'denied'" class="notification-warning">
          ⚠️ Notifications are blocked. Please enable them in your browser settings.
        </div>
      </div>
    </template>
    
    <div v-if="links.length" class="table-container">
      <table class="links-table">
        <thead>
          <tr>
            <th>Remote Node</th>
            <th>Node Information</th>
            <th class="hide-mobile">Status</th>
            <th class="hide-portrait">Direction</th>
            <th class="hide-portrait">Connected</th>
            <th class="hide-portrait">Mode</th>
            <th v-if="showIP" class="hide-portrait">IP Address</th>
            <th class="hide-portrait">Last Heard</th>
            <th class="hide-portrait">Total TX</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="l in sortedLinks" :key="l.node" :class="{ txing: l.current_tx || l.is_keyed }">
            <td class="node-num">
              <a v-if="l.node > 0" :href="`https://stats.allstarlink.org/stats/${l.node}`" target="_blank" rel="noopener noreferrer" class="node-link">
                {{ formatNodeNumber(l.node, l.node_callsign) }}
              </a>
              <span v-else>{{ formatNodeNumber(l.node, l.node_callsign) }}</span>
            </td>
            <td class="node-info">
              <div v-if="l.node_callsign" class="callsign">
                <a :href="`https://www.qrz.com/db/${l.node_callsign.toUpperCase()}`" target="_blank" rel="noopener noreferrer" class="callsign-link">
                  <span class="callsign-text">{{ l.node_callsign }}</span>
                </a>
              </div>
              <div v-if="l.node_description || l.node_location" class="node-details">
                <span v-if="l.node_description">{{ l.node_description }}</span>
                <span v-if="l.node_location" class="location">{{ l.node_location }}</span>
              </div>
              <div v-if="!l.node_callsign" class="loading">Loading...</div>
            </td>
            <td class="hide-mobile">
              <span class="status-badge" :class="{ active: l.current_tx || l.is_keyed }">
                {{ (l.current_tx || l.is_keyed) ? '● TX' : 'IDLE' }}
              </span>
            </td>
            <td class="hide-portrait">
              <span class="direction-badge" :class="l.direction?.toLowerCase()">
                {{ l.direction || '—' }}
              </span>
            </td>
            <td class="hide-portrait">{{ l.elapsed || formatSince(l.connected_since) }}</td>
            <td class="hide-portrait">
              <span class="mode-badge" :class="modeClass(l.mode)">
                {{ formatMode(l.mode) }}
              </span>
            </td>
            <td v-if="showIP" class="hide-portrait ip-addr">{{ l.ip || '—' }}</td>
            <td class="hide-portrait last-heard">{{ l.last_heard || (l.last_heard_at ? formatSince(l.last_heard_at) : 'Never') }}</td>
            <td class="hide-portrait">{{ formatDuration(l.total_tx_seconds) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="no-data">
      No active links
    </div>
  </Card>
</template>

<script setup>
import { computed, toRef, ref, watch, onMounted } from 'vue'
import Card from './Card.vue'
import { useAuthStore } from '../stores/auth'
import { useNodeLookup } from '../composables/useNodeLookup'

const props = defineProps({
  links: Array,
  status: Object,
  nowTick: Number
})

const authStore = useAuthStore()
const showIP = computed(() => authStore.isAdmin)

// Notification settings
const showNotificationSettings = ref(false)
const notificationsEnabled = ref(localStorage.getItem('txNotificationsEnabled') === 'true')
const notificationCooldown = ref(parseInt(localStorage.getItem('txNotificationCooldown') || '30'))
const notificationPermission = ref(typeof Notification !== 'undefined' ? Notification.permission : 'denied')
const lastNotificationTime = ref(0)
const wasTransmitting = ref(false)
const cooldownRemaining = ref(0)

// Sound settings
const soundEnabled = ref(localStorage.getItem('txSoundEnabled') === 'true')
const soundVolume = ref(parseInt(localStorage.getItem('txSoundVolume') || '80'))
const soundType = ref(localStorage.getItem('txSoundType') || 'two-tone')
const speechEnabled = ref(localStorage.getItem('txSpeechEnabled') === 'true')
let audioContext = null
let notificationSound = null

// Initialize audio context and load sound
function initAudio() {
  if (!audioContext) {
    audioContext = new (window.AudioContext || window.webkitAudioContext)()
  }
}

// Generate a notification tone (beep sound)
// Returns the duration of the sound in milliseconds
function playNotificationSound() {
  try {
    initAudio()
    
    // Resume audio context if suspended (browser autoplay policy)
    if (audioContext.state === 'suspended') {
      audioContext.resume()
    }
    
    const now = audioContext.currentTime
    const oscillator = audioContext.createOscillator()
    const gainNode = audioContext.createGain()
    
    // Connect nodes
    oscillator.connect(gainNode)
    gainNode.connect(audioContext.destination)
    
    // Set volume (0 to 1 scale)
    const volume = soundVolume.value / 100
    gainNode.gain.setValueAtTime(volume * 0.3, now)
    
    // Configure sound based on selected type
    oscillator.type = 'sine'
    let duration = 0.35 // Default duration in seconds
    
    switch (soundType.value) {
      case 'single-beep':
        // Simple 800Hz beep
        oscillator.frequency.setValueAtTime(800, now)
        gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.2)
        oscillator.start(now)
        oscillator.stop(now + 0.2)
        duration = 0.2
        break
        
      case 'two-tone':
        // Two beeps (800Hz -> 1000Hz)
        oscillator.frequency.setValueAtTime(800, now)
        oscillator.frequency.setValueAtTime(1000, now + 0.15)
        gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.35)
        oscillator.start(now)
        oscillator.stop(now + 0.35)
        duration = 0.35
        break
        
      case 'triple-beep':
        // Three quick beeps
        oscillator.frequency.setValueAtTime(800, now)
        oscillator.frequency.setValueAtTime(1000, now + 0.1)
        oscillator.frequency.setValueAtTime(800, now + 0.2)
        oscillator.frequency.setValueAtTime(1000, now + 0.3)
        gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.4)
        oscillator.start(now)
        oscillator.stop(now + 0.4)
        duration = 0.4
        break
        
      case 'ascending':
        // Ascending scale (600 -> 800 -> 1000 -> 1200Hz)
        oscillator.frequency.setValueAtTime(600, now)
        oscillator.frequency.exponentialRampToValueAtTime(1200, now + 0.5)
        gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.6)
        oscillator.start(now)
        oscillator.stop(now + 0.6)
        duration = 0.6
        break
        
      case 'descending':
        // Descending scale (1200 -> 1000 -> 800 -> 600Hz)
        oscillator.frequency.setValueAtTime(1200, now)
        oscillator.frequency.exponentialRampToValueAtTime(600, now + 0.5)
        gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.6)
        oscillator.start(now)
        oscillator.stop(now + 0.6)
        duration = 0.6
        break
        
      case 'chirp':
        // Quick chirp sound (800Hz -> 2000Hz very fast)
        oscillator.frequency.setValueAtTime(800, now)
        oscillator.frequency.exponentialRampToValueAtTime(2000, now + 0.15)
        gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.2)
        oscillator.start(now)
        oscillator.stop(now + 0.2)
        duration = 0.2
        break
        
      default:
        // Fallback to two-tone
        oscillator.frequency.setValueAtTime(800, now)
        oscillator.frequency.setValueAtTime(1000, now + 0.15)
        gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.35)
        oscillator.start(now)
        oscillator.stop(now + 0.35)
        duration = 0.35
    }
    
  logger.info('Played notification sound:', soundType.value, 'at volume:', soundVolume.value, 'duration:', duration)
    
    // Return duration in milliseconds (with small buffer)
    return Math.ceil(duration * 1000) + 100
  } catch (err) {
  logger.error('Failed to play notification sound:', err)
    return 0
  }
}

function speakNotification(text) {
  try {
    if ('speechSynthesis' in window) {
      // Cancel any ongoing speech
      window.speechSynthesis.cancel()
      
      const utterance = new SpeechSynthesisUtterance(text)
      utterance.rate = 1.0
      utterance.pitch = 1.0
      utterance.volume = soundVolume.value / 100
      
      window.speechSynthesis.speak(utterance)
  logger.info('Speaking:', text)
    } else {
  logger.error('Speech synthesis not supported')
    }
  } catch (err) {
  logger.error('Failed to speak notification:', err)
  }
}

function testSpeech() {
  const testText = 'This is a test of the speech notification system.'
  speakNotification(testText)
}

function onSoundToggle() {
  localStorage.setItem('txSoundEnabled', soundEnabled.value)
  if (soundEnabled.value) {
    // Play test sound when enabled
    playNotificationSound()
  }
}

function onSoundTypeChange() {
  localStorage.setItem('txSoundType', soundType.value)
  // Play the new sound when changed
  playNotificationSound()
}

function onSpeechToggle() {
  localStorage.setItem('txSpeechEnabled', speechEnabled.value)
  if (speechEnabled.value) {
    // Test speech when enabled
    testSpeech()
  }
}

function onVolumeChange() {
  localStorage.setItem('txSoundVolume', soundVolume.value)
}

// Update cooldown remaining every second
let cooldownInterval = null

function updateCooldownRemaining() {
  if (!notificationsEnabled.value || lastNotificationTime.value === 0) {
    cooldownRemaining.value = 0
    return
  }
  
  const now = Date.now()
  const cooldownMs = notificationCooldown.value * 1000
  const elapsed = now - lastNotificationTime.value
  const remaining = Math.max(0, cooldownMs - elapsed)
  
  cooldownRemaining.value = Math.ceil(remaining / 1000) // Convert to seconds
}

function formatCooldownTime(seconds) {
  if (seconds < 60) {
    return `${seconds}s`
  }
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}m ${secs}s`
}

onMounted(() => {
  // Check notification permission on mount
  if (typeof Notification !== 'undefined') {
    notificationPermission.value = Notification.permission
  }
  
  // Start cooldown timer
  cooldownInterval = setInterval(updateCooldownRemaining, 1000)
})

// Clean up interval on unmount
import { onUnmounted } from 'vue'
onUnmounted(() => {
  if (cooldownInterval) {
    clearInterval(cooldownInterval)
  }
})

// Card title showing local node info
const cardTitle = computed(() => {
  const nodeId = props.status?.node_id
  const nodeCallsign = props.status?.node_callsign
  
  if (!nodeId) return 'Active Links'
  
  if (nodeCallsign) {
    return `Active Links - Node ${nodeId} (${nodeCallsign})`
  }
  
  return `Active Links - Node ${nodeId}`
})

// Check if any link is currently transmitting
const anyLinkTransmitting = computed(() => {
  return props.links && props.links.some(link => link.current_tx || link.is_keyed)
})

// Request notification permission and update state
async function onNotificationToggle() {
  logger.info('Notification toggle:', notificationsEnabled.value)
  
  if (notificationsEnabled.value) {
    if (typeof Notification === 'undefined') {
      alert('Notifications are not supported in this browser')
      notificationsEnabled.value = false
      return
    }
    
  logger.info('Current permission:', Notification.permission)
    
    if (Notification.permission === 'default') {
  logger.info('Requesting notification permission...')
      const permission = await Notification.requestPermission()
  logger.info('Permission response:', permission)
      notificationPermission.value = permission
      
      if (permission !== 'granted') {
        alert('Notification permission was denied')
        notificationsEnabled.value = false
        return
      }
    } else if (Notification.permission === 'denied') {
      alert('Notifications are blocked. Please enable them in your browser settings.')
      notificationsEnabled.value = false
      return
    } else if (Notification.permission === 'granted') {
  logger.info('Permission already granted')
      // Test notification
      sendTestNotification()
    }
  }
  
  localStorage.setItem('txNotificationsEnabled', notificationsEnabled.value)
  logger.info('Saved notification setting:', notificationsEnabled.value)
}

function sendTestNotification() {
  logger.info('Test notification requested')
  logger.info('Notification support:', typeof Notification !== 'undefined')
  logger.info('Permission:', typeof Notification !== 'undefined' ? Notification.permission : 'N/A')
  
  if (typeof Notification === 'undefined') {
    alert('Notifications are not supported in this browser')
    return
  }
  
  if (Notification.permission !== 'granted') {
    alert('Please enable notifications first (permission: ' + Notification.permission + ')')
    return
  }
  
  try {
  logger.info('Creating test notification...')
    const notification = new Notification('✅ Test Notification', {
      body: 'Notifications are working! You will receive TX alerts.',
      icon: '/favicon.ico',
      tag: 'test-notification',
      requireInteraction: false
    })
    
  logger.info('Test notification created successfully')
    
    // Set lastNotificationTime to trigger cooldown display
    lastNotificationTime.value = Date.now()
    
    notification.onerror = (err) => {
  logger.error('Notification error event:', err)
    }
    
    notification.onshow = () => {
  logger.info('Notification shown successfully')
    }
    
    setTimeout(() => {
  logger.info('Closing test notification')
      notification.close()
    }, 5000)
  } catch (err) {
  logger.error('Failed to create test notification:', err)
    alert('Failed to create notification: ' + err.message)
  }
}

// Watch for cooldown changes
watch(notificationCooldown, (newValue) => {
  localStorage.setItem('txNotificationCooldown', newValue)
})

// Watch for TX state changes and send notifications
watch(anyLinkTransmitting, (isTransmitting) => {
  logger.info('TX state changed:', isTransmitting, 'Notifications enabled:', notificationsEnabled.value)
  logger.debug('Current lastNotificationTime:', lastNotificationTime.value)
  logger.debug('Current wasTransmitting:', wasTransmitting.value)
  
  if (!notificationsEnabled.value) {
    wasTransmitting.value = isTransmitting
    return
  }
  
  const now = Date.now()
  const cooldownMs = notificationCooldown.value * 1000
  
  // TX just started
  if (isTransmitting && !wasTransmitting.value) {
  logger.debug('TX started - checking if system was idle long enough')
    
    // Check if system has been idle (no TX) for longer than cooldown period
    const timeSinceLastTxStop = now - lastNotificationTime.value
    
  logger.debug('Now:', now)
  logger.debug('Last TX stop time:', lastNotificationTime.value)
  logger.debug('Time since last TX stop (ms):', timeSinceLastTxStop)
  logger.debug('Cooldown period (ms):', cooldownMs)
    
    // Send notification if:
    // 1. Never had TX before (lastNotificationTime === 0), OR
    // 2. System has been idle for longer than cooldown period
    if (lastNotificationTime.value === 0 || timeSinceLastTxStop >= cooldownMs) {
  logger.info('✅ System was idle long enough - sending notification')
      sendTxNotification()
    } else {
      const remainingCooldown = Math.ceil((cooldownMs - timeSinceLastTxStop) / 1000)
  logger.info('❌ System still considered busy (not idle long enough)')
  logger.info('Need to wait', remainingCooldown, 'more seconds after TX stops for system to be "idle"')
    }
  }
  
  // TX just stopped - record the stop time to start measuring idle period
  if (!isTransmitting && wasTransmitting.value) {
  logger.debug('TX stopped - recording stop time to measure idle period')
    // Record when TX stopped - cooldown period must pass before system is considered "idle"
    lastNotificationTime.value = now
  logger.debug('Set lastNotificationTime (TX stop time) to:', lastNotificationTime.value)
  logger.debug('System will be considered "idle" in', notificationCooldown.value, 'seconds')
  }
  
  wasTransmitting.value = isTransmitting
  logger.debug('Updated wasTransmitting to:', wasTransmitting.value)
})

function sendTxNotification() {
  logger.debug('sendTxNotification called')
  logger.debug('Notification available:', typeof Notification !== 'undefined')
  logger.debug('Notification permission:', typeof Notification !== 'undefined' ? Notification.permission : 'N/A')
  
  if (typeof Notification === 'undefined' || Notification.permission !== 'granted') {
  logger.error('Cannot send notification - permission not granted')
    return
  }
  
  // Get transmitting node info
  const txLink = props.links.find(link => link.current_tx || link.is_keyed)
  if (!txLink) {
  logger.error('No TX link found')
    return
  }
  
  const nodeName = txLink.node_callsign || `Node ${txLink.node}`
  const title = '🔴 TX Active'
  const body = `${nodeName} is now transmitting`
  
  logger.debug('Creating notification:', title, body)
  
  try {
    const notification = new Notification(title, {
      body,
      icon: '/favicon.ico',
      badge: '/favicon.ico',
      tag: 'tx-notification', // Reuse same notification
      requireInteraction: false,
      silent: false
    })
    
  logger.debug('Notification created successfully')
    
    // Play sound and/or speech based on settings
    // If both are enabled, play sound first, then speech after sound completes
    if (soundEnabled.value && speechEnabled.value) {
  logger.debug('Playing sound followed by speech')
      const soundDuration = playNotificationSound()
      setTimeout(() => {
        speakNotification(body)
      }, soundDuration)
    } else if (soundEnabled.value) {
  logger.debug('Playing notification sound only')
      playNotificationSound()
    } else if (speechEnabled.value) {
  logger.debug('Speaking notification only')
      speakNotification(body)
    }
    
    // Auto-close after 5 seconds
    setTimeout(() => notification.close(), 5000)
    
    // Focus window if notification is clicked
    notification.onclick = () => {
      window.focus()
      notification.close()
    }
  } catch (err) {
  logger.error('Failed to create notification:', err)
  }
}

// Enrich links with node information from astdb
const { enrichedLinks } = useNodeLookup(toRef(props, 'links'))

// Sort links: active talker(s) at top, then by total_tx_seconds (desc), then by connected duration (desc)
const sortedLinks = computed(() => {
  if (!enrichedLinks.value) return []
  const arr = [...enrichedLinks.value]
  const now = Date.now()
  return arr.sort((a, b) => {
    const aActive = !!(a.current_tx || a.is_keyed)
    const bActive = !!(b.current_tx || b.is_keyed)
    if (aActive && !bActive) return -1
    if (!aActive && bActive) return 1

    // Both same activity state: sort by total_tx_seconds (desc)
    const aTx = Number(a.total_tx_seconds || 0)
    const bTx = Number(b.total_tx_seconds || 0)
    if (aTx !== bTx) return bTx - aTx

    // Then by connected duration (longer connected first)
    const aConn = a.connected_since ? now - new Date(a.connected_since).getTime() : -1
    const bConn = b.connected_since ? now - new Date(b.connected_since).getTime() : -1
    if (aConn !== bConn) return bConn - aConn

    // Final fallback: most recently heard first
    const aTime = a.last_keyed_time || a.last_heard_at || a.connected_since
    const bTime = b.last_keyed_time || b.last_heard_at || b.connected_since
    if (!aTime && !bTime) return 0
    if (!aTime) return 1
    if (!bTime) return -1
    return new Date(bTime) - new Date(aTime)
  })
})

// Format node number - for negative numbers (text nodes), show callsign instead
// Handle duplicates by adding -1, -2, -3 suffix
const callsignCounts = computed(() => {
  const counts = {}
  const seen = {}

  sortedLinks.value.forEach(link => {
    if (link.node < 0 && link.node_callsign) {
      const callsign = link.node_callsign
      if (!counts[callsign]) {
        counts[callsign] = 0
        seen[callsign] = []
      }
      counts[callsign]++
      seen[callsign].push(link.node)
    }
  })

  return { counts, seen }
})

function formatNodeNumber(nodeNum, callsign) {
  // Positive numbers are regular AllStar nodes - show as-is
  if (nodeNum >= 0) {
    return nodeNum
  }

  // Negative numbers are hashed text nodes - show callsign
  if (callsign) {
    const { counts, seen } = callsignCounts.value

    // If only one instance, just show callsign
    if (counts[callsign] === 1) {
      return callsign
    }

    // Multiple instances - add suffix -1, -2, -3, etc.
    const index = seen[callsign].indexOf(nodeNum)
    return `${callsign}-${index + 1}`
  }

  // Fallback - shouldn't happen
  return nodeNum
}

function formatSince(ts) {
  if (!ts) return '—'
  const d = new Date(ts)
  const diff = (Date.now()-d.getTime())/1000
  if (diff < 60) return Math.floor(diff)+ 's ago'
  if (diff < 3600) return Math.floor(diff/60)+ 'm ago'
  if (diff < 86400) return Math.floor(diff/3600)+ 'h ago'
  return Math.floor(diff/86400)+'d ago'
}

function formatDuration(secs) {
  if (!secs || secs === 0) return '0s'
  const h = Math.floor(secs / 3600)
  const m = Math.floor((secs % 3600) / 60)
  const s = secs % 60
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

function formatMode(mode) {
  if (!mode) return '—'
  const modes = {
    'T': 'Transceive',
    'R': 'Receive',
    'C': 'Connecting',
    'M': 'Monitor'
  }
  return modes[mode] || mode
}

function modeClass(mode) {
  if (!mode) return ''
  return `mode-${mode.toLowerCase()}`
}
</script>

<style scoped>
.card-header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  width: 100%;
}

.card-header-content h3 {
  margin: 0;
  font-size: 1.25rem;
  color: var(--text-primary);
  font-weight: 600;
  flex: 1;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.settings-btn {
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 0.5rem;
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.settings-btn:hover {
  background: var(--bg-hover);
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}

.notification-settings {
  margin-top: 1rem;
  padding: 1rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
}

.setting-row {
  margin-bottom: 0.75rem;
}

.setting-row:last-child {
  margin-bottom: 0;
}

.setting-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-primary);
  font-size: 0.875rem;
  cursor: pointer;
}

.setting-label input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
}

.cooldown-select {
  margin-left: auto;
  padding: 0.375rem 0.75rem;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 0.875rem;
  cursor: pointer;
}

.cooldown-select:focus {
  outline: none;
  border-color: var(--accent-primary);
}

.sound-select {
  margin-left: auto;
  padding: 0.375rem 0.75rem;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 0.875rem;
  cursor: pointer;
}

.sound-select:focus {
  outline: none;
  border-color: var(--accent-primary);
}

.setting-help {
  margin: 0.5rem 0 0 0;
  font-size: 0.75rem;
  color: var(--text-muted);
  font-style: italic;
}

.notification-warning {
  margin-top: 0.75rem;
  padding: 0.75rem;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid #ef4444;
  border-radius: 4px;
  color: #ef4444;
  font-size: 0.875rem;
}

.button-row {
  display: flex;
  gap: 0.5rem;
}

.test-notification-btn {
  flex: 1;
  padding: 0.625rem 1rem;
  background: var(--accent-primary);
  border: none;
  border-radius: 6px;
  color: white;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.test-notification-btn.secondary {
  background: #6b7280;
}

.test-notification-btn:hover {
  background: var(--accent-hover);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.test-notification-btn.secondary:hover {
  background: #4b5563;
}

.test-notification-btn:active {
  transform: translateY(0);
}

.volume-control {
  display: flex !important;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
}

.volume-slider {
  flex: 1;
  height: 6px;
  border-radius: 3px;
  background: var(--border-color);
  outline: none;
  -webkit-appearance: none;
  appearance: none;
  cursor: pointer;
}

.volume-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--accent-primary);
  cursor: pointer;
  transition: all 0.2s;
}

.volume-slider::-webkit-slider-thumb:hover {
  background: var(--accent-hover);
  transform: scale(1.2);
}

.volume-slider::-moz-range-thumb {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--accent-primary);
  cursor: pointer;
  border: none;
  transition: all 0.2s;
}

.volume-slider::-moz-range-thumb:hover {
  background: var(--accent-hover);
  transform: scale(1.2);
}

.volume-value {
  min-width: 40px;
  text-align: right;
  font-size: 0.875rem;
  color: var(--text-secondary);
  font-weight: 600;
}

.cooldown-status {
  padding: 0.625rem 1rem;
  background: var(--accent-primary);
  border: none;
  border-radius: 6px;
  color: white;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.test-notification-btn:hover {
  background: var(--accent-hover);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.test-notification-btn:active {
  transform: translateY(0);
}

.cooldown-status {
  padding: 0.625rem;
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid var(--accent-primary);
  border-radius: 4px;
  color: var(--accent-primary);
  font-size: 0.875rem;
  text-align: center;
  font-weight: 500;
}

.tx-indicator {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.75rem;
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid #ef4444;
  border-radius: 6px;
  color: #ef4444;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  animation: tx-glow 2s ease-in-out infinite;
}

.tx-pulse {
  display: inline-block;
  width: 8px;
  height: 8px;
  background: #ef4444;
  border-radius: 50%;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.5;
    transform: scale(1.3);
  }
}

@keyframes tx-glow {
  0%, 100% {
    box-shadow: 0 0 5px rgba(239, 68, 68, 0.3);
  }
  50% {
    box-shadow: 0 0 15px rgba(239, 68, 68, 0.6);
  }
}

.table-container {
  overflow-x: auto;
}

.links-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.links-table th,
.links-table td {
  padding: 0.5rem 0.75rem;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}

.links-table thead th {
  background: var(--bg-tertiary);
  color: var(--text-label);
  text-transform: uppercase;
  font-size: 0.75rem;
  font-weight: 600;
  position: sticky;
  top: 0;
}

.links-table tbody tr {
  transition: background-color 0.2s;
}

.links-table tbody tr:hover {
  background: var(--bg-hover);
}

.links-table tbody tr.txing {
  animation: blink 1s linear infinite;
  background: var(--bg-hover);
}

@keyframes blink {
  50% { background: var(--bg-tertiary); }
}

.node-num {
  font-weight: 600;
  color: var(--accent-primary);
}

.node-num .node-link {
  color: var(--accent-primary);
  text-decoration: none;
  font-weight: 600;
  transition: color 0.2s;
}

.node-num .node-link:hover {
  color: var(--accent-hover);
  text-decoration: underline;
}

.node-info .callsign-link {
  color: #10b981;
  text-decoration: none;
  transition: color 0.2s;
}

.node-info .callsign-link:hover {
  color: #059669;
  text-decoration: underline;
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
  background: var(--bg-tertiary);
  color: var(--text-muted);
}

.status-badge.active {
  background: var(--error);
  color: #fff;
  animation: pulse-badge 1s infinite;
}

@keyframes pulse-badge {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.no-data {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  font-style: italic;
}

.mode-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
}

.mode-badge.mode-t {
  background: #059669;
  color: #fff;
}

.mode-badge.mode-r {
  background: #0284c7;
  color: #fff;
}

.mode-badge.mode-c {
  background: #ca8a04;
  color: #fff;
}

.mode-badge.mode-m {
  background: #6b7280;
  color: #fff;
}

.direction-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 600;
}

.direction-badge.in {
  background: #1e3a8a;
  color: #93c5fd;
}

.direction-badge.out {
  background: #7c2d12;
  color: #fdba74;
}

.ip-addr {
  font-family: 'Courier New', monospace;
  font-size: 0.8rem;
  color: #9ca3af;
}

.last-heard {
  font-weight: 500;
  color: #d1d5db;
}

.node-info {
  min-width: 200px;
}

.node-info .callsign {
  font-weight: 600;
  color: #10b981;
  font-size: 0.875rem;
  margin-bottom: 0.125rem;
}

.node-info .node-details {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  font-size: 0.75rem;
  color: #9ca3af;
}

.node-info .location {
  color: #60a5fa;
  font-style: italic;
}

.node-info .loading {
  color: #6b7280;
  font-size: 0.75rem;
  font-style: italic;
}

/* Responsive: Hide columns on mobile portrait */
@media (max-width: 767px) {
  .hide-mobile {
    display: none;
  }
}

/* Show all columns on landscape or larger screens */
@media (max-width: 767px) and (orientation: portrait) {
  .hide-portrait {
    display: none;
  }
}

/* Ensure columns are visible on landscape mobile or tablets+ */
@media (min-width: 768px), (orientation: landscape) {
  .hide-portrait {
    display: table-cell;
  }

  .hide-mobile {
    display: table-cell;
  }
}
</style>
