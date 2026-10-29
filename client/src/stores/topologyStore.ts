import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { DatabaseNode, ReplicationLink, TopologyGraph } from '@/types/topology'

export const useTopologyStore = defineStore('topology', () => {
  const graph = ref<TopologyGraph | null>(null)
  const selectedNodeId = ref<string | null>(null)
  const layoutDirection = ref<'TB' | 'LR'>('TB')
  const isLiveConnected = ref(false)
  const isPaused = ref(false)
  const lastUpdated = ref<Date | null>(null)

  const nodes = computed<DatabaseNode[]>(() => graph.value?.nodes || [])
  const links = computed<ReplicationLink[]>(() => graph.value?.links || [])

  const activePrimaries = computed(() =>
    nodes.value.filter((n) => n.role === 'PRIMARY' && n.status !== 'OFFLINE')
  )

  const hasSplitBrain = computed(() => {
    if (graph.value?.splitBrainDetected) return true
    return activePrimaries.value.length > 1
  })

  const clusterWarnings = computed<string[]>(() => graph.value?.warnings || [])

  const selectedNode = computed<DatabaseNode | null>(() => {
    if (!selectedNodeId.value) return null
    return nodes.value.find((n) => n.id === selectedNodeId.value) || null
  })

  function setGraph(newGraph: TopologyGraph) {
    if (isPaused.value) return

    graph.value = newGraph
    lastUpdated.value = new Date()

    // If currently selected node was removed, reset selection
    if (selectedNodeId.value && !newGraph.nodes.some((n) => n.id === selectedNodeId.value)) {
      selectedNodeId.value = null
    }
  }

  function selectNode(id: string | null) {
    selectedNodeId.value = id
  }

  function toggleLayoutDirection() {
    layoutDirection.value = layoutDirection.value === 'TB' ? 'LR' : 'TB'
  }

  function togglePause() {
    isPaused.value = !isPaused.value
  }

  function setLiveConnected(connected: boolean) {
    isLiveConnected.value = connected
  }

  return {
    graph,
    nodes,
    links,
    selectedNodeId,
    selectedNode,
    layoutDirection,
    isLiveConnected,
    isPaused,
    lastUpdated,
    activePrimaries,
    hasSplitBrain,
    clusterWarnings,
    setGraph,
    selectNode,
    toggleLayoutDirection,
    togglePause,
    setLiveConnected,
  }
})

