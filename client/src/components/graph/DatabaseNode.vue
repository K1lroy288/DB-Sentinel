<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import type { DatabaseNode as DatabaseNodeType } from '@/types/topology'
import { useTopologyStore } from '@/stores/topologyStore'
import { useReplicationTraffic } from '@/composables/useReplicationTraffic'
import { 
  Database, 
  GitFork, 
  Copy, 
  Check, 
  Lock
} from 'lucide-vue-next'
import { ref } from 'vue'

const props = defineProps<{
  id: string
  data: DatabaseNodeType
  targetPosition?: Position
  sourcePosition?: Position
}>()

const store = useTopologyStore()
const { formatBytes } = useReplicationTraffic()
const copied = ref(false)

const isSelected = computed(() => store.selectedNodeId === props.id)

const roleStyles = computed(() => {
  switch (props.data.role) {
    case 'PRIMARY':
      return {
        badge: 'bg-indigo-500/20 text-indigo-300 border-indigo-500/30',
        border: 'border-indigo-500/50 hover:border-indigo-400',
        glow: 'shadow-indigo-500/10',
        icon: Database,
      }
    case 'CASCADING_REPLICA':
      return {
        badge: 'bg-amber-500/20 text-amber-300 border-amber-500/30',
        border: 'border-amber-500/50 hover:border-amber-400',
        glow: 'shadow-amber-500/10',
        icon: GitFork,
      }
    case 'REPLICA':
    default:
      return {
        badge: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30',
        border: 'border-emerald-500/40 hover:border-emerald-400',
        glow: 'shadow-emerald-500/10',
        icon: Copy,
      }
  }
})

const statusIndicator = computed(() => {
  switch (props.data.status) {
    case 'HEALTHY':
      return { color: 'bg-emerald-400', ring: 'ring-emerald-400/30', label: 'Healthy' }
    case 'LAGGING':
      return { color: 'bg-amber-400', ring: 'ring-amber-400/30', label: 'Lagging' }
    case 'CRITICAL':
      return { color: 'bg-red-400', ring: 'ring-red-400/30', label: 'Critical' }
    case 'OFFLINE':
    default:
      return { color: 'bg-gray-500', ring: 'ring-gray-500/20', label: 'Offline' }
  }
})

function copyHost(e: Event) {
  e.stopPropagation()
  navigator.clipboard.writeText(props.data.id)
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
}

function handleSelect() {
  store.selectNode(props.id)
}
</script>

<template>
  <div
    class="w-[280px] bg-slate-900/95 backdrop-blur-md rounded-xl border p-4 shadow-xl transition-all duration-200 cursor-pointer"
    :class="[
      roleStyles.border,
      roleStyles.glow,
      isSelected ? 'ring-2 ring-blue-400 !border-blue-400 shadow-blue-500/20' : '',
      data.status === 'OFFLINE' ? 'opacity-65 grayscale-[30%]' : ''
    ]"
    @click="handleSelect"
  >
    <!-- Vue Flow Connector Handles -->
    <Handle
      type="target"
      :position="targetPosition || Position.Top"
      class="!w-3 !h-3 !bg-slate-300 !border-2 !border-slate-800 transition-colors"
    />
    <Handle
      type="source"
      :position="sourcePosition || Position.Bottom"
      class="!w-3 !h-3 !bg-blue-500 !border-2 !border-slate-800 transition-colors"
    />

    <!-- Header: Role badge & Health Dot -->
    <div class="flex items-center justify-between gap-2 mb-3">
      <div
        class="flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold border"
        :class="roleStyles.badge"
      >
        <component :is="roleStyles.icon" class="w-3.5 h-3.5" />
        <span>{{ data.role }}</span>
      </div>

      <!-- Health Status Dot -->
      <div class="flex items-center gap-1.5" :title="statusIndicator.label">
        <span class="relative flex h-2.5 w-2.5">
          <span
            v-if="data.status !== 'OFFLINE'"
            class="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75"
            :class="statusIndicator.color"
          ></span>
          <span
            class="relative inline-flex rounded-full h-2.5 w-2.5"
            :class="statusIndicator.color"
          ></span>
        </span>
        <span class="text-[11px] font-medium text-slate-400">{{ statusIndicator.label }}</span>
      </div>
    </div>

    <!-- Node Identity (Host & Port) -->
    <div class="flex items-center justify-between group mb-3">
      <div class="font-mono text-sm font-bold text-slate-100 tracking-tight truncate" :title="data.id">
        {{ data.id }}
      </div>
      <button
        class="text-slate-400 hover:text-slate-200 p-1 rounded hover:bg-slate-800/80 transition-colors"
        :title="copied ? 'Copied!' : 'Copy Address'"
        @click="copyHost"
      >
        <Check v-if="copied" class="w-3.5 h-3.5 text-emerald-400" />
        <Copy v-else class="w-3.5 h-3.5" />
      </button>
    </div>

    <!-- Metrics Bar -->
    <div class="grid grid-cols-2 gap-2 text-xs pt-2 border-t border-slate-800/80">
      <!-- PostgreSQL Version -->
      <div class="flex flex-col">
        <span class="text-[10px] uppercase font-semibold text-slate-400">Version</span>
        <span class="text-slate-200 font-mono text-[11px] truncate" :title="data.version">
          {{ data.version || 'PostgreSQL' }}
        </span>
      </div>

      <!-- Replication Lag (for standbys) or Status -->
      <div class="flex flex-col text-right">
        <span class="text-[10px] uppercase font-semibold text-slate-400">
          {{ data.role === 'PRIMARY' ? 'Access' : 'Repl Lag' }}
        </span>
        <div v-if="data.role === 'PRIMARY'" class="flex items-center justify-end gap-1 text-slate-300 font-medium text-[11px]">
          <span class="text-emerald-400 font-semibold">Read/Write</span>
        </div>
        <div v-else class="font-mono text-[11px] flex items-center justify-end gap-1">
          <span
            :class="[
              data.metrics.replicationLagSeconds > 10 ? 'text-red-400 font-bold' :
              data.metrics.replicationLagSeconds >= 1 ? 'text-amber-400' : 'text-emerald-400 font-medium'
            ]"
          >
            {{ data.metrics.replicationLagSeconds > 0 ? `${data.metrics.replicationLagSeconds.toFixed(1)}s` : '0s' }}
          </span>
          <span class="text-slate-500">/</span>
          <span class="text-slate-400">{{ formatBytes(data.metrics.replicationLagBytes) }}</span>
        </div>
      </div>
    </div>

    <!-- Read-only badge if replica -->
    <div v-if="data.readOnly" class="mt-2.5 flex items-center gap-1 text-[10px] text-slate-400 font-medium">
      <Lock class="w-3 h-3 text-slate-500" />
      <span>Read-Only Standby</span>
    </div>
  </div>
</template>
