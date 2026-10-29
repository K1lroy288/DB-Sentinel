import dagre from '@dagrejs/dagre'
import { Position } from '@vue-flow/core'

export function useTopologyLayout() {
  function layout<T extends { id: string; position: { x: number; y: number } }>(
    nodes: T[],
    edges: Array<{ source: string; target: string }>,
    direction: 'TB' | 'LR' = 'TB'
  ): T[] {
    const dagreGraph = new dagre.graphlib.Graph()
    dagreGraph.setDefaultEdgeLabel(() => ({}))

    const isHorizontal = direction === 'LR'
    dagreGraph.setGraph({
      rankdir: direction,
      nodesep: isHorizontal ? 90 : 110,
      ranksep: isHorizontal ? 140 : 120,
    })

    const nodeWidth = 280
    const nodeHeight = 165

    for (const node of nodes) {
      dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight })
    }

    for (const edge of edges) {
      dagreGraph.setEdge(edge.source, edge.target)
    }

    dagre.layout(dagreGraph)

    return nodes.map((node) => {
      const nodeWithPosition = dagreGraph.node(node.id)
      return {
        ...node,
        targetPosition: isHorizontal ? Position.Left : Position.Top,
        sourcePosition: isHorizontal ? Position.Right : Position.Bottom,
        position: {
          x: nodeWithPosition.x - nodeWidth / 2,
          y: nodeWithPosition.y - nodeHeight / 2,
        },
      }
    })
  }

  return { layout }
}
