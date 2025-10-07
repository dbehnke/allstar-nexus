import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useNodeStore = defineStore('node', () => {
  const status = ref(null)
  const links = ref([])
  const talker = ref([])
  const topLinks = ref([])
  const nowTick = ref(Date.now())
  // pending stop timers: node -> timer id
  const pendingStops = new Map()
  // track server session to detect restarts
  const lastSessionStart = ref(null)

  function handleWSMessage(msg) {
    if (msg.messageType === 'STATUS_UPDATE') {
      // Detect server restart by checking session_start
      if (msg.data.session_start && lastSessionStart.value && msg.data.session_start !== lastSessionStart.value) {
        // Server restarted - clear talker log
        talker.value = []
        console.log('Server restarted, cleared talker log')
      }
      lastSessionStart.value = msg.data.session_start

      status.value = msg.data
      if (msg.data.links_detailed) links.value = msg.data.links_detailed
    } else if (msg.messageType === 'TALKER_LOG_SNAPSHOT') {
      // Full talker log snapshot from server (on connect or periodic refresh)
      // Replace entire talker log with enriched server data
      // Note: events may not have a node field at all (omitempty), so keep all events
      talker.value = Array.isArray(msg.data) ? msg.data : []
      console.log('Received talker log snapshot:', talker.value.length, 'events')
    } else if (msg.messageType === 'TALKER_EVENT') {
      try {
        const incoming = msg.data
        
        // Skip events without a valid node number (node 0 or missing)
        if (!incoming.node || incoming.node === 0) {
          return
        }
        
        const now = new Date(incoming.at).getTime()
        
        // Check last 5 events for duplicates (same node+kind within 2s)
        const recentEvents = talker.value.slice(-5)
        for (const recent of recentEvents) {
          if (recent.kind === incoming.kind && recent.node === incoming.node) {
            const recentTs = new Date(recent.at).getTime()
            if (Math.abs(now - recentTs) < 2000) {
              console.log('Skipping duplicate talker event:', incoming.kind, incoming.node)
              return
            }
          }
        }

        // If this is a STOP without a known start, buffer it briefly to allow a delayed START to arrive
        if (incoming.kind === 'TX_STOP') {
          const nodeId = incoming.node || 0
          const hasRecentStart = talker.value.some(t => (t.node || 0) === nodeId && t.kind === 'TX_START' && (new Date(incoming.at).getTime() - new Date(t.at).getTime()) < 30000)
          if (!hasRecentStart) {
            // set a 1s timer to push the STOP unless a START arrives in the meantime
            if (pendingStops.has(nodeId)) {
              clearTimeout(pendingStops.get(nodeId))
            }
            const tid = setTimeout(() => {
              talker.value.push(incoming)
              if (talker.value.length > 100) talker.value.shift()
              pendingStops.delete(nodeId)
            }, 1000)
            pendingStops.set(nodeId, tid)
            return
          }
        }

        // Normal push
        talker.value.push(incoming)
        if (talker.value.length > 100) talker.value.shift()
        // If this is a START, cancel any pending STOP timers for the same node
        if (incoming.kind === 'TX_START') {
          const nodeId = incoming.node || 0
          if (pendingStops.has(nodeId)) {
            clearTimeout(pendingStops.get(nodeId))
            pendingStops.delete(nodeId)
          }
        }
      } catch (e) {
        talker.value.push(msg.data)
        if (talker.value.length > 100) talker.value.shift()
      }
    } else if (msg.messageType === 'LINK_ADDED') {
      for (const add of msg.data) {
        if (!links.value.find(l => l.node === add.node)) links.value.push(add)
      }
    } else if (msg.messageType === 'LINK_REMOVED') {
      for (const id of msg.data) {
        const idx = links.value.findIndex(l => l.node === id)
        if (idx >= 0) links.value.splice(idx, 1)
      }
    } else if (msg.messageType === 'LINK_TX') {
      updateLinkTX([msg.data])
    } else if (msg.messageType === 'LINK_TX_BATCH') {
      updateLinkTX(msg.data)
    }
  }

  function updateLinkTX(events) {
    for (const e of events) {
      const li = links.value.find(l => l.node === e.node)
      if (li) {
        if (e.kind === 'START') {
          li.current_tx = true
          li.last_tx_start = e.last_tx_start || e.at
        } else if (e.kind === 'STOP') {
          li.current_tx = false
          li.last_tx_end = e.last_tx_end || e.at
          li.total_tx_seconds = e.total_tx_seconds
        }
      }
    }
  }

  function setTopLinks(data) {
    topLinks.value = data
  }

  function startTickTimer() {
    setInterval(() => {
      nowTick.value = Date.now()
    }, 1000)
  }

  function loadTalkerHistory(events) {
    // Load historical talker events (from API on page load)
    // Keep all events - node field may be omitted entirely due to omitempty
    talker.value = events || []
  }

  return {
    status,
    links,
    talker,
    topLinks,
    nowTick,
    handleWSMessage,
    setTopLinks,
    startTickTimer,
    loadTalkerHistory
  }
})
