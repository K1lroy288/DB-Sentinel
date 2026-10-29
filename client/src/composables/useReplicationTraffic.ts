import type { ChannelStatus } from '@/types/topology'

export function useReplicationTraffic() {
  function getEdgeColor(lagSeconds: number, status: ChannelStatus): string {
    if (status === 'DISCONNECTED') {
      return '#EF4444' // Red
    }
    if (status === 'RECOVERING' || lagSeconds > 10) {
      return '#EF4444' // Red
    }
    if (lagSeconds >= 1) {
      return '#F59E0B' // Amber / Yellow
    }
    return '#10B981' // Green
  }

  function getStreamAnimationClass(lagSeconds: number, status: ChannelStatus): string {
    if (status === 'DISCONNECTED') {
      return 'edge-stream-broken'
    }
    if (status === 'RECOVERING' || lagSeconds > 10) {
      return 'edge-stream-slow'
    }
    if (lagSeconds >= 1) {
      return 'edge-stream-medium'
    }
    return 'edge-stream-fast'
  }

  function formatBytes(bytes: number): string {
    if (bytes <= 0) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(1024))
    const val = bytes / Math.pow(1024, i)
    return `${val.toFixed(val < 10 && i > 0 ? 1 : 0)} ${units[i]}`
  }

  function formatLag(seconds: number, bytes: number): string {
    if (seconds <= 0 && bytes <= 0) {
      return '0s / 0 B'
    }
    const timeStr = seconds < 1 && seconds > 0 
      ? `${(seconds * 1000).toFixed(0)} ms` 
      : `${seconds.toFixed(1)} s`
    return `${timeStr} (${formatBytes(bytes)})`
  }

  return {
    getEdgeColor,
    getStreamAnimationClass,
    formatBytes,
    formatLag,
  }
}

