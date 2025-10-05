import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useNodeStore = defineStore('node', () => {
  const status = ref(null)
  const links = ref([])
  const talker = ref([])
  const topLinks = ref([])
  const nowTick = ref(Date.now())

  function handleWSMessage(msg) {
    if (msg.messageType === 'STATUS_UPDATE') {
      status.value = msg.data
      if (msg.data.links_detailed) links.value = msg.data.links_detailed
    } else if (msg.messageType === 'TALKER_EVENT') {
      talker.value.push(msg.data)
      if (talker.value.length > 100) talker.value.shift()
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

  return {
    status,
    links,
    talker,
    topLinks,
    nowTick,
    handleWSMessage,
    setTopLinks,
    startTickTimer
  }
})
