import { onMounted, onUnmounted } from 'vue'
import { useTopologyStore } from '@/stores/topologyStore'
import type { TopologyGraph } from '@/types/topology'

export function useWebSocket() {
  const store = useTopologyStore()
  let socket: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let mockTimer: ReturnType<typeof setInterval> | null = null

  function connect() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws`

    try {
      socket = new WebSocket(wsUrl)

      socket.onopen = () => {
        store.setLiveConnected(true)
        if (mockTimer) {
          clearInterval(mockTimer)
          mockTimer = null
        }
      }

      socket.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data) as TopologyGraph
          if (payload && payload.nodes) {
            store.setGraph(payload)
          }
        } catch (e) {
          console.error('Failed to parse WebSocket topology message', e)
        }
      }

      socket.onclose = () => {
        store.setLiveConnected(false)
        scheduleReconnect()
      }

      socket.onerror = () => {
        store.setLiveConnected(false)
        // If WebSocket server is not available (e.g. standalone frontend dev), run mock data
        if (!mockTimer && store.nodes.length === 0) {
          startMockDataStream()
        }
      }
    } catch (e) {
      store.setLiveConnected(false)
      scheduleReconnect()
    }
  }

  function scheduleReconnect() {
    if (reconnectTimer) clearTimeout(reconnectTimer)
    reconnectTimer = setTimeout(() => {
      connect()
    }, 3000)
  }

  function startMockDataStream() {
    console.log('[DB-Sentinel] Starting Dev Mock Topology Stream...')
    let tick = 0

    const mockGraph = (): TopologyGraph => {
      tick++
      const lagSeconds = (Math.sin(tick * 0.4) + 1) * 1.5 // oscillates 0s - 3s
      const lagBytes = Math.floor(lagSeconds * 1024 * 512)

      return {
        clusterName: 'postgres-cluster',
        timestamp: Date.now(),
        nodes: [
          {
            id: 'postgres-master:5432',
            hostname: 'postgres-master',
            role: 'PRIMARY',
            status: 'HEALTHY',
            version: 'PostgreSQL 18.0',
            readOnly: false,
            metrics: {
              replicationLagBytes: 0,
              replicationLagSeconds: 0,
              transactionsPerSecond: 142.5,
              cpuLoadPercent: 18.4,
            },
            tags: {
              system_identifier: '7412891902847291823',
            },
          },
          {
            id: 'postgres-replica-1:5432',
            hostname: 'postgres-replica-1',
            role: 'CASCADING_REPLICA',
            status: 'HEALTHY',
            version: 'PostgreSQL 18.0',
            readOnly: true,
            metrics: {
              replicationLagBytes: Math.floor(lagBytes * 0.3),
              replicationLagSeconds: lagSeconds * 0.2,
              transactionsPerSecond: 0,
              cpuLoadPercent: 8.2,
            },
            tags: {
              system_identifier: '7412891902847291823',
            },
          },
          {
            id: 'postgres-replica-2:5432',
            hostname: 'postgres-replica-2',
            role: 'REPLICA',
            status: lagSeconds > 2.5 ? 'LAGGING' : 'HEALTHY',
            version: 'PostgreSQL 18.0',
            readOnly: true,
            metrics: {
              replicationLagBytes: lagBytes,
              replicationLagSeconds: lagSeconds,
              transactionsPerSecond: 0,
              cpuLoadPercent: 12.1,
            },
            tags: {
              system_identifier: '7412891902847291823',
            },
          },
        ],
        links: [
          {
            id: 'repl-master-replica1',
            sourceNodeId: 'postgres-master:5432',
            targetNodeId: 'postgres-replica-1:5432',
            type: 'ASYNC',
            status: 'ACTIVE',
            positionDetails: {
              sourceLsn: `0/${(3000000 + tick * 100).toString(16)}`,
              appliedLsn: `0/${(3000000 + tick * 95).toString(16)}`,
              sentLsn: `0/${(3000000 + tick * 100).toString(16)}`,
            },
          },
          {
            id: 'repl-replica1-replica2',
            sourceNodeId: 'postgres-replica-1:5432',
            targetNodeId: 'postgres-replica-2:5432',
            type: 'ASYNC',
            status: 'ACTIVE',
            positionDetails: {
              sourceLsn: `0/${(3000000 + tick * 95).toString(16)}`,
              appliedLsn: `0/${(3000000 + tick * 80).toString(16)}`,
              sentLsn: `0/${(3000000 + tick * 95).toString(16)}`,
            },
          },
        ],
      }
    }

    store.setGraph(mockGraph())
    mockTimer = setInterval(() => {
      store.setGraph(mockGraph())
    }, 2000)
  }

  onMounted(() => {
    connect()
  })

  onUnmounted(() => {
    if (socket) socket.close()
    if (reconnectTimer) clearTimeout(reconnectTimer)
    if (mockTimer) clearInterval(mockTimer)
  })

  return {
    startMockDataStream,
  }
}
