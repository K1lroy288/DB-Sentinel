<script setup lang="ts">
import { computed } from 'vue'
import { useTopologyStore } from '@/stores/topologyStore'
import { useReplicationTraffic } from '@/composables/useReplicationTraffic'
import { 
  X, 
  Server, 
  Activity, 
  Hash, 
  ArrowRight
} from 'lucide-vue-next'

const store = useTopologyStore()
const { formatBytes } = useReplicationTraffic()

const node = computed(() => store.selectedNode)

const connectedLinks = computed(() => {
  if (!node.value) return []
  return store.links.filter(
    (l) => l.sourceNodeId === node.value?.id || l.targetNodeId === node.value?.id
  )
})

function close() {
  store.selectNode(null)
}
</script>

<template>
  <transition
    enter-active-class="transform transition ease-in-out duration-300"
    enter-from-class="translate-x-full"
    enter-to-class="translate-x-0"
    leave-active-class="transform transition ease-in-out duration-300"
    leave-from-class="translate-x-0"
    leave-to-class="translate-x-full"
  >
    <div
      v-if="node"
      class="fixed inset-y-0 right-0 z-30 w-96 bg-slate-900/95 border-l border-slate-800 shadow-2xl backdrop-blur-xl p-6 flex flex-col justify-between overflow-y-auto"
    >
      <div>
        <!-- Drawer Header -->
        <div class="flex items-center justify-between pb-4 border-b border-slate-800">
          <div class="flex items-center gap-2.5">
            <div class="p-2 rounded-lg bg-blue-500/10 text-blue-400 border border-blue-500/20">
              <Server class="w-4 h-4" />
            </div>
            <div>
              <h2 class="text-sm font-bold text-white tracking-tight">Node Inspector</h2>
              <p class="text-[11px] font-mono text-slate-400 truncate max-w-[200px]" :title="node.id">
                {{ node.id }}
              </p>
            </div>
          </div>
          <button
            class="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
            @click="close"
          >
            <X class="w-4 h-4" />
          </button>
        </div>

        <!-- Role & Health Overview Card -->
        <div class="my-5 p-4 rounded-xl bg-slate-800/40 border border-slate-700/50 flex flex-col gap-3">
          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-400">Cluster Role</span>
            <span class="font-bold text-white px-2 py-0.5 rounded bg-slate-700/60 font-mono">
              {{ node.role }}
            </span>
          </div>

          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-400">Health Status</span>
            <span
              class="font-semibold text-xs font-mono"
              :class="[
                node.status === 'HEALTHY' ? 'text-emerald-400' :
                node.status === 'LAGGING' ? 'text-amber-400' :
                node.status === 'CRITICAL' ? 'text-red-400' : 'text-gray-400'
              ]"
            >
              {{ node.status }}
            </span>
          </div>

          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-400">PostgreSQL Engine</span>
            <span class="text-slate-200 font-mono text-[11px] truncate max-w-[160px]" :title="node.version">
              {{ node.version || 'Unknown' }}
            </span>
          </div>
        </div>

        <!-- Detailed Metrics Section -->
        <div class="flex flex-col gap-4">
          <h3 class="text-xs font-semibold text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
            <Activity class="w-3.5 h-3.5 text-blue-400" />
            <span>Replication Telemetry</span>
          </h3>

          <div class="grid grid-cols-2 gap-3 text-xs">
            <div class="p-3 rounded-xl bg-slate-800/30 border border-slate-800 flex flex-col gap-1">
              <span class="text-[10px] text-slate-400 font-medium">Replication Lag</span>
              <span class="font-mono text-sm font-bold"
                :class="node.metrics.replicationLagSeconds > 10 ? 'text-red-400' : node.metrics.replicationLagSeconds >= 1 ? 'text-amber-400' : 'text-emerald-400'"
              >
                {{ node.metrics.replicationLagSeconds > 0 ? `${node.metrics.replicationLagSeconds.toFixed(2)}s` : '0s' }}
              </span>
              <span class="text-[10px] text-slate-400 font-mono">
                {{ formatBytes(node.metrics.replicationLagBytes) }}
              </span>
            </div>

            <div class="p-3 rounded-xl bg-slate-800/30 border border-slate-800 flex flex-col gap-1">
              <span class="text-[10px] text-slate-400 font-medium">Access Permissions</span>
              <span class="font-mono text-sm font-bold text-white">
                {{ node.readOnly ? 'Read-Only' : 'Read/Write' }}
              </span>
              <span class="text-[10px] text-slate-400">
                {{ node.readOnly ? 'Standby Mode' : 'Accepts Transactions' }}
              </span>
            </div>
          </div>

          <!-- System Identifier Tag (if available) -->
          <div v-if="node.tags?.system_identifier" class="p-3 rounded-xl bg-slate-800/30 border border-slate-800 flex flex-col gap-1 text-xs">
            <div class="flex items-center gap-1 text-[10px] font-semibold text-slate-400 uppercase">
              <Hash class="w-3 h-3 text-indigo-400" />
              <span>System Identifier</span>
            </div>
            <span class="font-mono text-xs text-indigo-300 break-all">
              {{ node.tags.system_identifier }}
            </span>
          </div>

          <!-- Connected Replication Links -->
          <div class="flex flex-col gap-2 mt-2">
            <h4 class="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
              Connected Channels ({{ connectedLinks.length }})
            </h4>

            <div v-if="connectedLinks.length === 0" class="text-xs text-slate-400 italic">
              No active replication channels found for this node.
            </div>

            <div
              v-for="link in connectedLinks"
              :key="link.id"
              class="p-2.5 rounded-lg bg-slate-800/40 border border-slate-700/60 flex flex-col gap-1.5 text-xs font-mono"
            >
              <div class="flex items-center justify-between text-[11px]">
                <span class="text-slate-300 truncate max-w-[120px]" :title="link.sourceNodeId">
                  {{ link.sourceNodeId }}
                </span>
                <ArrowRight class="w-3.5 h-3.5 text-slate-500" />
                <span class="text-slate-300 truncate max-w-[120px]" :title="link.targetNodeId">
                  {{ link.targetNodeId }}
                </span>
              </div>

              <div class="flex items-center justify-between text-[10px] text-slate-400 pt-1 border-t border-slate-800">
                <span class="text-blue-400 font-semibold">{{ link.type }}</span>
                <span :class="link.status === 'ACTIVE' ? 'text-emerald-400 font-medium' : 'text-amber-400'">
                  {{ link.status }}
                </span>
              </div>

              <div v-if="link.positionDetails.appliedLsn" class="text-[10px] text-slate-400 flex items-center justify-between">
                <span>Applied LSN:</span>
                <span class="text-slate-300 font-bold">{{ link.positionDetails.appliedLsn }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Drawer Footer -->
      <div class="pt-4 border-t border-slate-800 flex justify-end">
        <button
          class="px-4 py-2 rounded-lg text-xs font-medium bg-slate-800 text-slate-200 hover:bg-slate-700 transition-colors"
          @click="close"
        >
          Close Inspector
        </button>
      </div>
    </div>
  </transition>
</template>
