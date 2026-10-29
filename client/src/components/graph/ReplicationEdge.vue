<script setup lang="ts">
import { computed } from 'vue'
import { BaseEdge, EdgeLabelRenderer, getSmoothStepPath, Position } from '@vue-flow/core'
import type { ReplicationLink } from '@/types/topology'
import { useReplicationTraffic } from '@/composables/useReplicationTraffic'

const props = defineProps<{
  id: string
  sourceX: number
  sourceY: number
  targetX: number
  targetY: number
  sourcePosition: Position
  targetPosition: Position
  data?: ReplicationLink
  markerEnd?: string
}>()

const { getEdgeColor, getStreamAnimationClass } = useReplicationTraffic()

const path = computed(() => {
  return getSmoothStepPath({
    sourceX: props.sourceX,
    sourceY: props.sourceY,
    sourcePosition: props.sourcePosition,
    targetX: props.targetX,
    targetY: props.targetY,
    targetPosition: props.targetPosition,
    borderRadius: 16,
  })
})

const lagSeconds = computed(() => {
  return 0 // Standby node metrics hold the lag, link can display type and status
})

const edgeColor = computed(() => {
  return getEdgeColor(lagSeconds.value, props.data?.status || 'ACTIVE')
})

const streamClass = computed(() => {
  return getStreamAnimationClass(lagSeconds.value, props.data?.status || 'ACTIVE')
})
</script>

<template>
  <!-- Background Glow & Animated Path -->
  <BaseEdge
    :id="id"
    :path="path[0]"
    :marker-end="markerEnd"
    :style="{ stroke: edgeColor }"
    :class="['vue-flow__edge-path', streamClass]"
  />

  <!-- Edge Label Renderer: Type and Status Badge -->
  <EdgeLabelRenderer>
    <div
      :style="{
        transform: `translate(-50%, -50%) translate(${path[1]}px,${path[2]}px)`,
      }"
      class="nodrag nopan absolute pointer-events-auto"
    >
      <div
        class="flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[10px] font-mono font-semibold border backdrop-blur-md shadow-md bg-slate-900/90 border-slate-700/80 text-slate-300"
      >
        <span
          class="w-1.5 h-1.5 rounded-full"
          :style="{ backgroundColor: edgeColor }"
        ></span>
        <span>{{ data?.type || 'ASYNC' }}</span>
        <span v-if="data?.status && data.status !== 'ACTIVE'" class="text-amber-400">
          ({{ data.status }})
        </span>
      </div>
    </div>
  </EdgeLabelRenderer>
</template>
