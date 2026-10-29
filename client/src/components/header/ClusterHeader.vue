<script setup lang="ts">
import { computed } from 'vue'
import { useTopologyStore } from '@/stores/topologyStore'
import { 
  ShieldCheck, 
  Wifi, 
  WifiOff, 
  Clock, 
  Layers
} from 'lucide-vue-next'

const store = useTopologyStore()

const clusterName = computed(() => store.graph?.clusterName || 'postgres-cluster')

const stats = computed(() => {
  const total = store.nodes.length
  const healthy = store.nodes.filter((n) => n.status === 'HEALTHY').length
  const lagging = store.nodes.filter((n) => n.status === 'LAGGING').length
  const offline = store.nodes.filter((n) => n.status === 'OFFLINE').length
  return { total, healthy, lagging, offline }
})

const formattedTime = computed(() => {
  if (!store.lastUpdated) return 'Syncing...'
  return store.lastUpdated.toLocaleTimeString()
})
</script>

<template>
  <header class="h-16 border-b border-slate-800 bg-slate-900/90 backdrop-blur-md px-6 flex items-center justify-between z-20">
    <!-- Brand / Title -->
    <div class="flex items-center gap-3">
      <div class="p-2 rounded-xl bg-blue-500/10 border border-blue-500/20 text-blue-400">
        <ShieldCheck class="w-5 h-5" />
      </div>
      <div>
        <div class="flex items-center gap-2">
          <h1 class="text-sm font-bold text-white tracking-tight">DB-Sentinel</h1>
          <span class="px-2 py-0.5 rounded text-[10px] font-mono font-semibold bg-slate-800 text-slate-300 border border-slate-700/60">
            {{ clusterName }}
          </span>
        </div>
        <p class="text-[11px] text-slate-400">Database Replication Topology & Sentinel Health</p>
      </div>
    </div>

    <!-- Cluster Overview Stats -->
    <div class="flex items-center gap-4 text-xs font-medium">
      <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-800/60 border border-slate-700/50">
        <Layers class="w-3.5 h-3.5 text-slate-400" />
        <span class="text-slate-300">Nodes:</span>
        <span class="font-bold text-white">{{ stats.total }}</span>
        <span class="text-slate-500">|</span>
        <span class="text-emerald-400 font-semibold">{{ stats.healthy }} OK</span>
        <span v-if="stats.lagging > 0" class="text-amber-400 font-semibold">{{ stats.lagging }} Lag</span>
        <span v-if="stats.offline > 0" class="text-gray-400 font-semibold">{{ stats.offline }} Off</span>
      </div>

      <!-- Sync Time & WS Status -->
      <div class="flex items-center gap-2 text-slate-400">
        <Clock class="w-3.5 h-3.5 text-slate-500" />
        <span class="text-[11px] font-mono">{{ formattedTime }}</span>
      </div>

      <!-- Live Indicator -->
      <div class="flex items-center gap-1.5 text-xs font-semibold px-2.5 py-1 rounded-full border"
        :class="store.isLiveConnected ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30' : 'bg-amber-500/10 text-amber-400 border-amber-500/30'"
      >
        <Wifi v-if="store.isLiveConnected" class="w-3.5 h-3.5 animate-pulse" />
        <WifiOff v-else class="w-3.5 h-3.5" />
        <span>{{ store.isLiveConnected ? 'Live' : 'Polling' }}</span>
      </div>
    </div>
  </header>
</template>
