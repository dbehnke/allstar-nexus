import { watch } from 'vue'
import { useNodeStore } from '../stores/node'
import { useTxNotifications } from './useTxNotifications'
import { logger } from '../utils/logger'

// Keep singleton map of per-source notification handlers
const handlers = new Map()

function ensureHandler(sourceNodeID) {
  if (!handlers.has(sourceNodeID)) {
    try {
      const notif = useTxNotifications(sourceNodeID)
      handlers.set(sourceNodeID, { notif, lastTxKeyed: undefined })
      logger.debug('[GlobalTxNotif] created handler for source', sourceNodeID)
    } catch (err) {
      logger.debug('[GlobalTxNotif] failed to create handler for', sourceNodeID, err)
    }
  }
  return handlers.get(sourceNodeID)
}

function findTransmittingAdjacent(entry) {
  try {
    const adj = entry && (entry.adjacentNodes || entry.adjacent_nodes) || {}
    for (const v of Object.values(adj)) {
      if (v && (v.IsTransmitting || v.is_transmitting || v.tx)) {
        // Normalize shape expected by useTxNotifications
        return { NodeID: v.NodeID || v.node || v.Node || 0, Callsign: v.Callsign || v.callsign || '' }
      }
    }
  } catch {}
  return { NodeID: entry && (entry.sourceNodeID || entry.source_node_id) || 0, Callsign: '' }
}

export function initGlobalTxNotifications() {
  const store = useNodeStore()
  // Watch for source node keying updates; trigger notifications on TX state edges
  watch(() => store.sourceNodes, (val) => {
    try {
      const map = (val && val.value) || val || {}
      for (const [k, entry] of Object.entries(map)) {
        const sid = parseInt(k, 10)
        const h = ensureHandler(sid)
        if (!h) continue
        const txKeyed = !!(entry && (entry.txKeyed || entry.tx_keyed))
        if (h.lastTxKeyed === undefined) {
          h.lastTxKeyed = txKeyed
          continue
        }
        if (txKeyed !== h.lastTxKeyed) {
          h.lastTxKeyed = txKeyed
          const txNode = findTransmittingAdjacent(entry)
          try {
            h.notif.watchTxState(txKeyed, txNode)
          } catch (err) { logger.debug('[GlobalTxNotif] watchTxState failed', err) }
        }
      }
    } catch (err) { logger.debug('[GlobalTxNotif] watch error', err) }
  }, { deep: true, immediate: true })
}
