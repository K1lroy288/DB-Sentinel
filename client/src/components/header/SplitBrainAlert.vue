<script setup lang="ts">
import { useTopologyStore } from '@/stores/topologyStore'
import { AlertOctagon, AlertTriangle } from 'lucide-vue-next'

const store = useTopologyStore()
</script>

<template>
  <div v-if="store.hasSplitBrain || store.clusterWarnings.length > 0" class="flex flex-col gap-1 px-6 py-2 bg-red-950/80 border-b border-red-500/30 backdrop-blur-md">
    <!-- Split-Brain Alert -->
    <div v-if="store.hasSplitBrain" class="flex items-center gap-2.5 text-red-300 text-xs font-semibold">
      <AlertOctagon class="w-4 h-4 text-red-400 animate-pulse flex-shrink-0" />
      <span>
        CRITICAL SPLIT-BRAIN DETECTED: {{ store.activePrimaries.length }} active PRIMARY nodes detected simultaneously! Immediate intervention required.
      </span>
    </div>

    <!-- Cluster-wide Warnings -->
    <div
      v-for="(warning, idx) in store.clusterWarnings"
      :key="idx"
      class="flex items-center gap-2 text-amber-300 text-xs font-medium"
    >
      <AlertTriangle class="w-3.5 h-3.5 text-amber-400 flex-shrink-0" />
      <span>{{ warning }}</span>
    </div>
  </div>
</template>

