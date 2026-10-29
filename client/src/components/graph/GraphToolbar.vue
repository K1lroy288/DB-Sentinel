<script setup lang="ts">
import { useTopologyStore } from '@/stores/topologyStore'
import { useVueFlow } from '@vue-flow/core'
import { 
  ArrowDownUp, 
  ArrowLeftRight, 
  Maximize2, 
  Pause, 
  Play, 
  RefreshCw 
} from 'lucide-vue-next'

const emit = defineEmits<{
  (e: 'relayout'): void
}>()

const store = useTopologyStore()
const { fitView } = useVueFlow()

function handleFitView() {
  fitView({ padding: 0.2, duration: 400 })
}
</script>

<template>
  <div class="absolute bottom-6 left-6 z-10 flex items-center gap-1.5 p-1.5 rounded-xl bg-slate-900/90 border border-slate-800 backdrop-blur-md shadow-2xl">
    <!-- Direction Switch (TB / LR) -->
    <button
      class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-slate-300 hover:text-white hover:bg-slate-800/80 transition-colors"
      :title="`Switch layout to ${store.layoutDirection === 'TB' ? 'Horizontal (Left-to-Right)' : 'Vertical (Top-to-Bottom)'}`"
      @click="store.toggleLayoutDirection(); emit('relayout')"
    >
      <ArrowLeftRight v-if="store.layoutDirection === 'TB'" class="w-3.5 h-3.5 text-blue-400" />
      <ArrowDownUp v-else class="w-3.5 h-3.5 text-blue-400" />
      <span>{{ store.layoutDirection === 'TB' ? 'Top-Down' : 'Left-Right' }}</span>
    </button>

    <div class="w-px h-4 bg-slate-800"></div>

    <!-- Fit View -->
    <button
      class="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800/80 transition-colors"
      title="Fit View"
      @click="handleFitView"
    >
      <Maximize2 class="w-3.5 h-3.5" />
    </button>

    <!-- Relayout -->
    <button
      class="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800/80 transition-colors"
      title="Re-calculate Layout"
      @click="emit('relayout')"
    >
      <RefreshCw class="w-3.5 h-3.5" />
    </button>

    <div class="w-px h-4 bg-slate-800"></div>

    <!-- Pause/Play Stream -->
    <button
      class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-colors"
      :class="store.isPaused ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30' : 'text-slate-400 hover:text-white hover:bg-slate-800/80'"
      :title="store.isPaused ? 'Resume live updates' : 'Pause live updates'"
      @click="store.togglePause"
    >
      <Play v-if="store.isPaused" class="w-3.5 h-3.5" />
      <Pause v-else class="w-3.5 h-3.5" />
      <span>{{ store.isPaused ? 'Paused' : 'Live' }}</span>
    </button>
  </div>
</template>

