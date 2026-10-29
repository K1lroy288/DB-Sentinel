<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { VueFlow, useVueFlow, MarkerType } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { MiniMap } from '@vue-flow/minimap'
import DatabaseNode from './DatabaseNode.vue'
import ReplicationEdge from './ReplicationEdge.vue'
import GraphToolbar from './GraphToolbar.vue'
import { useTopologyStore } from '@/stores/topologyStore'
import { useTopologyLayout } from '@/composables/useTopologyLayout'

const store = useTopologyStore()
const { layout } = useTopologyLayout()
const { fitView } = useVueFlow()

const flowNodes = ref<any[]>([])
const flowEdges = ref<any[]>([])

function updateFlowElements(triggerFit = false) {
  // 1. Build Vue Flow node items
  const currentNodes = flowNodes.value
  const rawNodes = store.nodes.map((n) => {
    // Preserve existing coordinates if available
    const existing = currentNodes.find((fn: { id: string }) => fn.id === n.id)
    return {
      id: n.id,
      type: 'databaseNode',
      position: existing ? existing.position : { x: 0, y: 0 },
      data: n,
    }
  })

  // 2. Build Vue Flow edge items
  const rawEdges = store.links.map((link) => {
    return {
      id: link.id,
      source: link.sourceNodeId,
      target: link.targetNodeId,
      type: 'replicationEdge',
      data: link,
      markerEnd: MarkerType.ArrowClosed,
    }
  })

  // 3. If layout is missing or nodes count changed, run Dagre auto-layout
  const needLayout = flowNodes.value.length !== rawNodes.length || flowNodes.value.some((fn) => fn.position.x === 0 && fn.position.y === 0)
  
  if (needLayout) {
    flowNodes.value = layout(rawNodes, rawEdges, store.layoutDirection)
    flowEdges.value = rawEdges
    if (triggerFit) {
      nextTick(() => {
        fitView({ padding: 0.25, duration: 400 })
      })
    }
  } else {
    // Soft update metrics in-place so positions don't jump
    for (const rawNode of rawNodes) {
      const target = flowNodes.value.find((fn: { id: string }) => fn.id === rawNode.id)
      if (target) {
        target.data = rawNode.data
      }
    }
    flowEdges.value = rawEdges
  }
}

function handleForceRelayout() {
  const rawNodes = (flowNodes.value as any[]).map((n) => ({ ...n }))
  flowNodes.value = layout(rawNodes, flowEdges.value, store.layoutDirection)
  nextTick(() => {
    fitView({ padding: 0.25, duration: 400 })
  })
}

watch(
  () => [store.nodes, store.links],
  () => {
    updateFlowElements(flowNodes.value.length === 0)
  },
  { deep: true, immediate: true }
)

watch(
  () => store.layoutDirection,
  () => {
    handleForceRelayout()
  }
)

function getMiniMapColor(node: any) {
  const role = node.data?.role
  if (role === 'PRIMARY') return '#6366F1'
  if (role === 'CASCADING_REPLICA') return '#F59E0B'
  return '#10B981'
}
</script>

<template>
  <div class="relative w-full h-full overflow-hidden">
    <VueFlow
      v-model:nodes="flowNodes"
      v-model:edges="flowEdges"
      :default-viewport="{ zoom: 1 }"
      :min-zoom="0.2"
      :max-zoom="2"
      fit-view-on-init
      class="w-full h-full"
    >
      <!-- Background Dot Grid -->
      <Background :gap="24" :size="1.5" color="#1E293B" />

      <!-- Custom Node Template -->
      <template #node-databaseNode="customNodeProps">
        <DatabaseNode
          :id="customNodeProps.id"
          :data="customNodeProps.data"
          :target-position="customNodeProps.targetPosition"
          :source-position="customNodeProps.sourcePosition"
        />
      </template>

      <!-- Custom Edge Template -->
      <template #edge-replicationEdge="customEdgeProps">
        <ReplicationEdge v-bind="customEdgeProps" />
      </template>

      <!-- Minimap in bottom-right corner -->
      <MiniMap
        :node-color="getMiniMapColor"
        :node-stroke-width="2"
        class="!bottom-6 !right-6 !m-0 shadow-2xl"
      />
    </VueFlow>

    <!-- Canvas Toolbar Controls -->
    <GraphToolbar @relayout="handleForceRelayout" />
  </div>
</template>
