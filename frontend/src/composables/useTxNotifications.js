import { ref, watch, computed, onMounted, onUnmounted } from 'vue'

/**
 * Composable for TX notifications per source node
 * Manages notification settings, sound, speech, and cooldown timers
 * Settings are stored per source node ID in localStorage
 */
export function useTxNotifications(sourceNodeID) {
  const storagePrefix = `txNotif_${sourceNodeID}_`

  // Notification settings
  const notificationsEnabled = ref(false)
  const soundEnabled = ref(true)
  const speechEnabled = ref(false)
  const soundType = ref('two-tone')
  const soundVolume = ref(80)
  const notificationCooldown = ref(30)
  const notificationPermission = ref('default')

  // State tracking
  const lastNotificationTime = ref(0)
  const wasTransmitting = ref(false)
  const cooldownRemaining = ref(0)
  const showSettings = ref(false)
  const initialized = ref(false)

  let cooldownInterval = null
  let audioContext = null
  const audioSuspended = ref(false)
  // Lightweight buffer to hold a single pending event if calls arrive before onMounted initializes
  // We only need to hold the most recent start/stop event; older events are not useful.
  let pendingEvent = null // { isTransmitting: boolean, txNode: object }

// Initialize settings synchronously so watchers have correct state immediately
try {
  notificationsEnabled.value = localStorage.getItem(`${storagePrefix}enabled`) === 'true'
  soundEnabled.value = localStorage.getItem(`${storagePrefix}sound`) !== 'false'
  speechEnabled.value = localStorage.getItem(`${storagePrefix}speech`) === 'true'
  soundType.value = localStorage.getItem(`${storagePrefix}soundType`) || 'two-tone'
  soundVolume.value = parseInt(localStorage.getItem(`${storagePrefix}volume`) || '80')
  notificationCooldown.value = parseInt(localStorage.getItem(`${storagePrefix}cooldown`) || '30')

  if (typeof Notification !== 'undefined') {
    notificationPermission.value = Notification.permission
  }
  console.debug('[TxNotif] init state:', {
    sourceNodeID,
    notificationsEnabled: notificationsEnabled.value,
    notificationPermission: notificationPermission.value,
    soundEnabled: soundEnabled.value,
    speechEnabled: speechEnabled.value
  })
} catch (err) {
  console.error('[TxNotif] Failed to read settings from localStorage during init:', err)
}

  // Load non-blocking runtime pieces on mount (timers, audio init when needed)
  onMounted(() => {
    // Start cooldown timer
    cooldownInterval = setInterval(updateCooldownRemaining, 1000)
    // mark as ready for callers
    initialized.value = true

    // Check whether audio is suspended (so UI can show an explicit enable button)
    try {
      checkAudioSuspended()
    } catch (err) {
      console.debug('[TxNotif] checkAudioSuspended failed on mount', err)
    }

    // If there was a pending event captured before initialization, process it now
    if (pendingEvent) {
      try {
        console.debug('[TxNotif] Replaying pending event on mount:', {
          time: new Date().toISOString(),
          sourceNodeID,
          pendingEvent: {
            isTransmitting: pendingEvent.isTransmitting,
            txNode: pendingEvent.txNode ? { NodeID: pendingEvent.txNode.NodeID, Callsign: pendingEvent.txNode.Callsign } : null
          }
        })
        watchTxState(pendingEvent.isTransmitting, pendingEvent.txNode)
      } catch (err) {
        console.error('[TxNotif] Failed to replay pending event:', err)
      }
      pendingEvent = null
    }
  })

// If notifications were previously enabled but browser permission is not granted,
// surface the settings UI so the user can request permission (must be user-initiated).
if (notificationsEnabled.value && typeof Notification !== 'undefined' && notificationPermission.value === 'default') {
  // show the settings panel so the user can click to request permission
  try {
    showSettings.value = true
  } catch (err) {
    // defensive: if called before Vue refs are ready, ignore
    console.debug('[TxNotif] showSettings not ready yet')
  }

  // Keep audio suspended state updated when sound or speech preferences change
  watch([soundEnabled, speechEnabled], () => {
    try {
      checkAudioSuspended()
    } catch (err) {
      console.debug('[TxNotif] checkAudioSuspended failed on preference change', err)
    }
  })
}

// Called when the user opens the settings UI (direct user gesture)
// Resume audio context (unblock audio) and, if notifications are enabled and
// permission is default, prompt for permission now (user gesture).
async function openSettings() {
  // toggle settings panel
  showSettings.value = !showSettings.value

  // Try to resume audio context on user gesture so future play calls succeed
  try {
    initAudio()
    if (audioContext && audioContext.state === 'suspended') {
      await audioContext.resume()
      console.debug('[TxNotif] AudioContext resumed via openSettings user gesture')
    }
  } catch (err) {
    console.debug('[TxNotif] openSettings: audio resume failed or not available', err)
  }

  // If notifications are enabled in settings and browser permission is still default,
  // request permission now because this function is invoked from a user gesture.
  try {
    if (notificationsEnabled.value && typeof Notification !== 'undefined' && notificationPermission.value === 'default') {
      const permission = await Notification.requestPermission()
      notificationPermission.value = permission
      console.debug('[TxNotif] openSettings: requested notification permission ->', permission)

      if (permission === 'granted') {
        // persist enabled
        localStorage.setItem(`${storagePrefix}enabled`, 'true')
        showSettings.value = false
        // Send a one-off test notification so the user immediately sees it working
        try {
          console.debug('[TxNotif] openSettings: sending immediate test notification')
          sendTestNotification()
        } catch (err) {
          console.error('[TxNotif] openSettings: failed to send immediate test notification', err)
        }
      } else if (permission === 'denied') {
        notificationsEnabled.value = false
        localStorage.setItem(`${storagePrefix}enabled`, 'false')
        // leave settings open so user sees the warning
      }
    }
  } catch (err) {
    console.error('[TxNotif] openSettings: permission request failed', err)
  }
}

// Allow the UI to request permission explicitly (must be called from a user gesture)
async function requestPermission() {
  if (typeof Notification === 'undefined') {
    alert('Notifications are not supported in this browser')
    return
  }

  try {
    const permission = await Notification.requestPermission()
    notificationPermission.value = permission

    if (permission === 'granted') {
      // keep notifications enabled and persist
      notificationsEnabled.value = true
      localStorage.setItem(`${storagePrefix}enabled`, 'true')
      showSettings.value = false
    } else {
      // user denied; turn off and persist
      notificationsEnabled.value = false
      localStorage.setItem(`${storagePrefix}enabled`, 'false')
      alert('Notification permission was denied')
    }
  } catch (err) {
    console.error('[TxNotif] requestPermission failed:', err)
    alert('Failed to request notification permission: ' + (err && err.message))
  }
}

  // Cleanup
  onUnmounted(() => {
    if (cooldownInterval) {
      clearInterval(cooldownInterval)
    }
  })

  // Update cooldown remaining time
  function updateCooldownRemaining() {
    if (lastNotificationTime.value === 0) {
      cooldownRemaining.value = 0
      return
    }

    const now = Date.now()
    const elapsed = Math.floor((now - lastNotificationTime.value) / 1000)
    const remaining = Math.max(0, notificationCooldown.value - elapsed)
    cooldownRemaining.value = remaining
  }

  // Initialize audio context
  function initAudio() {
    if (!audioContext) {
      audioContext = new (window.AudioContext || window.webkitAudioContext)()
    }
  }

  // Check whether audioContext is suspended (used to show enable audio UI)
  function checkAudioSuspended() {
    try {
      initAudio()
      if (audioContext && audioContext.state === 'suspended') {
        audioSuspended.value = true
      } else {
        audioSuspended.value = false
      }
    } catch (err) {
      console.debug('[TxNotif] checkAudioSuspended error', err)
      audioSuspended.value = false
    }
  }

  // Open/resume audio from a user gesture and play a quick test sound/speech
  async function openAudio() {
    try {
      initAudio()
      if (audioContext && audioContext.state === 'suspended') {
        await audioContext.resume()
        console.debug('[TxNotif] AudioContext resumed via openAudio user gesture')
      }

      // Play a short test sound and/or speech if enabled
      if (soundEnabled.value) {
        const dur = playNotificationSound()
        if (speechEnabled.value) {
          setTimeout(() => {
            speakNotification(`Audio enabled for node ${sourceNodeID}`)
          }, dur)
        }
      } else if (speechEnabled.value) {
        speakNotification(`Audio enabled for node ${sourceNodeID}`)
      }

      // update suspended flag
      checkAudioSuspended()
    } catch (err) {
      console.error('[TxNotif] openAudio failed:', err)
      alert('Failed to enable audio: ' + (err && err.message))
    }
  }

  // Play notification sound based on selected type
  function playNotificationSound() {
    try {
      initAudio()

      if (audioContext.state === 'suspended') {
        audioContext.resume()
      }

      const now = audioContext.currentTime
      const oscillator = audioContext.createOscillator()
      const gainNode = audioContext.createGain()

      oscillator.connect(gainNode)
      gainNode.connect(audioContext.destination)

      const volume = soundVolume.value / 100
      gainNode.gain.setValueAtTime(volume * 0.3, now)

      oscillator.type = 'sine'
      let duration = 0.35

      switch (soundType.value) {
        case 'single-beep':
          oscillator.frequency.setValueAtTime(800, now)
          gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.2)
          oscillator.start(now)
          oscillator.stop(now + 0.2)
          duration = 0.2
          break

        case 'two-tone':
          oscillator.frequency.setValueAtTime(800, now)
          oscillator.frequency.setValueAtTime(1000, now + 0.15)
          gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.35)
          oscillator.start(now)
          oscillator.stop(now + 0.35)
          duration = 0.35
          break

        case 'triple-beep':
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
          oscillator.frequency.setValueAtTime(600, now)
          oscillator.frequency.exponentialRampToValueAtTime(1200, now + 0.5)
          gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.6)
          oscillator.start(now)
          oscillator.stop(now + 0.6)
          duration = 0.6
          break

        case 'descending':
          oscillator.frequency.setValueAtTime(1200, now)
          oscillator.frequency.exponentialRampToValueAtTime(600, now + 0.5)
          gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.6)
          oscillator.start(now)
          oscillator.stop(now + 0.6)
          duration = 0.6
          break

        case 'chirp':
          oscillator.frequency.setValueAtTime(800, now)
          oscillator.frequency.exponentialRampToValueAtTime(2000, now + 0.15)
          gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.2)
          oscillator.start(now)
          oscillator.stop(now + 0.2)
          duration = 0.2
          break

        default:
          oscillator.frequency.setValueAtTime(800, now)
          oscillator.frequency.setValueAtTime(1000, now + 0.15)
          gainNode.gain.exponentialRampToValueAtTime(0.01, now + 0.35)
          oscillator.start(now)
          oscillator.stop(now + 0.35)
          duration = 0.35
      }

      console.log('[TxNotif] Played sound:', soundType.value, 'volume:', soundVolume.value)
      return Math.ceil(duration * 1000) + 100
    } catch (err) {
      console.error('[TxNotif] Failed to play sound:', err)
      return 0
    }
  }

  // Speak notification using Web Speech API
  function speakNotification(text) {
    try {
      if ('speechSynthesis' in window) {
        window.speechSynthesis.cancel()

        const utterance = new SpeechSynthesisUtterance(text)
        utterance.rate = 1.0
        utterance.pitch = 1.0
        utterance.volume = soundVolume.value / 100

        window.speechSynthesis.speak(utterance)
        console.log('[TxNotif] Speaking:', text)
      } else {
        console.error('[TxNotif] Speech synthesis not supported')
      }
    } catch (err) {
      console.error('[TxNotif] Failed to speak:', err)
    }
  }

  // Request notification permission
  async function onNotificationToggle() {
    if (notificationsEnabled.value) {
      if (typeof Notification === 'undefined') {
        alert('Notifications are not supported in this browser')
        notificationsEnabled.value = false
        return
      }

      if (Notification.permission === 'default') {
        const permission = await Notification.requestPermission()
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
      }
    }

    localStorage.setItem(`${storagePrefix}enabled`, notificationsEnabled.value)
  }

  // Send test notification
  function sendTestNotification() {
    if (typeof Notification === 'undefined' || Notification.permission !== 'granted') {
      alert('Please enable notifications first')
      return
    }

    try {
      const notification = new Notification('✅ Test Notification', {
        body: `Notifications are working for Node ${sourceNodeID}!`,
        icon: '/favicon.ico',
        tag: `test-notification-${sourceNodeID}`,
        requireInteraction: false
      })

      lastNotificationTime.value = Date.now()

      // Play sound and/or speech
      if (soundEnabled.value) {
        const soundDuration = playNotificationSound()
        if (speechEnabled.value) {
          setTimeout(() => {
            speakNotification(`Test notification for node ${sourceNodeID}`)
          }, soundDuration)
        }
      } else if (speechEnabled.value) {
        speakNotification(`Test notification for node ${sourceNodeID}`)
      }

      setTimeout(() => notification.close(), 5000)
    } catch (err) {
      console.error('[TxNotif] Failed to send test notification:', err)
      alert('Failed to create notification: ' + err.message)
    }
  }

  // Send TX notification with node info
  function sendTxNotification(txNode) {
    console.debug('[TxNotif] sendTxNotification: Notification.permission=', (typeof Notification !== 'undefined') ? Notification.permission : 'n/a')
    if (typeof Notification === 'undefined' || Notification.permission !== 'granted') {
      console.debug('[TxNotif] sendTxNotification: notifications unsupported or not granted; skipping')
      return
    }

    const nodeName = txNode.Callsign || `Node ${txNode.NodeID}`
    const title = '🔴 TX Active'
    const body = `${nodeName} is now transmitting on ${sourceNodeID}`

    try {
      const notification = new Notification(title, {
        body,
        icon: '/favicon.ico',
        tag: `tx-notification-${sourceNodeID}`,
        requireInteraction: false
      })

      console.log('[TxNotif] Sent notification:', title, body)

      // Play sound and/or speech
      if (soundEnabled.value) {
        const soundDuration = playNotificationSound()
        if (speechEnabled.value) {
          setTimeout(() => {
            speakNotification(`${nodeName} is transmitting`)
          }, soundDuration)
        }
      } else if (speechEnabled.value) {
        speakNotification(`${nodeName} is transmitting`)
      }

      setTimeout(() => notification.close(), 8000)
    } catch (err) {
      console.error('[TxNotif] Failed to send TX notification:', err)
    }
  }

  // Watch for changes and save to localStorage
  watch(soundEnabled, (val) => localStorage.setItem(`${storagePrefix}sound`, val))
  watch(speechEnabled, (val) => localStorage.setItem(`${storagePrefix}speech`, val))
  watch(soundType, (val) => localStorage.setItem(`${storagePrefix}soundType`, val))
  watch(soundVolume, (val) => localStorage.setItem(`${storagePrefix}volume`, val))
  watch(notificationCooldown, (val) => localStorage.setItem(`${storagePrefix}cooldown`, val))

  // Format cooldown time
  function formatCooldownTime(seconds) {
    if (seconds < 60) return `${seconds}s`
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}m ${secs}s`
  }

  // Test speech
  function testSpeech() {
    speakNotification(`This is a test of the speech notification system for node ${sourceNodeID}`)
  }

  // Monitor TX state changes
  function watchTxState(isTransmitting, txNode) {
    console.debug('[TxNotif] watchTxState called:', {
      time: new Date().toISOString(),
      sourceNodeID,
      isTransmitting,
      wasTransmitting: wasTransmitting.value,
      notificationsEnabled: notificationsEnabled.value,
      txNode: txNode ? { NodeID: txNode.NodeID, Callsign: txNode.Callsign } : null
    })

    // If composable hasn't completed onMounted initialization yet, buffer the latest event
    if (!initialized.value) {
      pendingEvent = { isTransmitting, txNode }
      console.debug('[TxNotif] Composable not initialized yet; buffering event:', {
        time: new Date().toISOString(),
        sourceNodeID,
        bufferedEvent: {
          isTransmitting: pendingEvent.isTransmitting,
          txNode: pendingEvent.txNode ? { NodeID: pendingEvent.txNode.NodeID, Callsign: pendingEvent.txNode.Callsign } : null
        }
      })
      // Still update wasTransmitting so state remains coherent
      wasTransmitting.value = isTransmitting
      return
    }

    if (!notificationsEnabled.value) {
      console.debug('[TxNotif] Notifications disabled, ignoring')
      wasTransmitting.value = isTransmitting
      return
    }

    const now = Date.now()
    const cooldownMs = notificationCooldown.value * 1000

    // TX just started
    if (isTransmitting && !wasTransmitting.value) {
      const timeSinceLastTxStop = now - lastNotificationTime.value

      console.debug('[TxNotif] TX started!', {
        lastNotificationTime: lastNotificationTime.value,
        timeSinceLastTxStop,
        cooldownMs,
        willNotify: lastNotificationTime.value === 0 || timeSinceLastTxStop >= cooldownMs
      })

      // Send notification if system was idle long enough
      if (lastNotificationTime.value === 0 || timeSinceLastTxStop >= cooldownMs) {
        console.debug('[TxNotif] Sending notification for:', txNode)
        sendTxNotification(txNode)
      } else {
        console.debug('[TxNotif] Skipping notification - cooldown active')
      }
    }

    // TX just stopped - record stop time
    if (!isTransmitting && wasTransmitting.value) {
      console.debug('[TxNotif] TX stopped, recording stop time')
      lastNotificationTime.value = now
    }

    wasTransmitting.value = isTransmitting
  }

  return {
    // Settings
    notificationsEnabled,
    soundEnabled,
    speechEnabled,
    soundType,
    soundVolume,
    notificationCooldown,
    notificationPermission,
    showSettings,

    // State
    cooldownRemaining,

    // Actions
    onNotificationToggle,
    sendTestNotification,
    playNotificationSound,
    testSpeech,
    formatCooldownTime,
    openSettings,
    audioSuspended,
    openAudio,
    watchTxState
  }
}
