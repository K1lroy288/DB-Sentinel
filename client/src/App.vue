<script setup lang="ts">
import ClusterHeader from '@/components/header/ClusterHeader.vue'
import SplitBrainAlert from '@/components/header/SplitBrainAlert.vue'
import TopologyCanvas from '@/components/graph/TopologyCanvas.vue'
import NodeDetailDrawer from '@/components/drawer/NodeDetailDrawer.vue'
import { useWebSocket } from '@/composables/useWebSocket'

// Initializes WebSocket listener (or mock data in standalone frontend dev)
useWebSocket()
</script>

<template>
  <div class="flex flex-col w-screen h-screen bg-dark-900 text-slate-100 overflow-hidden font-sans">
    <!-- Top Header -->
    <ClusterHeader />

    <!-- Split-Brain & Warnings Banner -->
    <SplitBrainAlert />

    <!-- Main Topology Canvas -->
    <main class="flex-1 relative w-full h-full">
      <TopologyCanvas />
    </main>

    <!-- Side Inspection Drawer -->
    <NodeDetailDrawer />
  </div>
</template>

<style>
/* Global resets */
html, body, #app {
  width: 100%;
  height: 100%;
  margin: 0;
  padding: 0;
}
</style>

