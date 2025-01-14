import { formatBytes } from '@/utils';
import { useTheme } from 'styled-components';
import { type Edge, type Node } from '@xyflow/react';
import { EDGE_TYPES, NODE_TYPES, OVERVIEW_ENTITY_TYPES, STATUSES, WorkloadId, type OverviewMetricsResponse } from '@/types';
import nodeConfig from './node-config.json';

interface Params {
  nodes: Node[];
  metrics?: OverviewMetricsResponse;
  containerHeight: number;
}

const { nodeHeight, framePadding } = nodeConfig;

const createEdge = (edgeId: string, params?: { label?: string; isMultiTarget?: boolean; isError?: boolean; animated?: boolean }): Edge => {
  const theme = useTheme();

  const { label, isMultiTarget, isError, animated } = params || {};
  const [sourceNodeId, targetNodeId] = edgeId.split('-to-');

  return {
    id: edgeId,
    type: !!label ? EDGE_TYPES.LABELED : 'default',
    source: sourceNodeId,
    target: targetNodeId,
    animated,
    data: { label, isMultiTarget, isError },
    style: { stroke: isError ? theme.colors.dark_red : theme.colors.border },
  };
};

export const buildEdges = ({ nodes, metrics, containerHeight }: Params) => {
  const edges: Edge[] = [];
  const actionNodeId = nodes.find(({ id: nodeId }) => ['action-frame', 'action-add'].includes(nodeId))?.id;

  nodes.forEach(({ type: nodeType, id: nodeId, data: { type: entityType, id: entityId, status }, position }) => {
    if (nodeType === NODE_TYPES.EDGED && entityType === OVERVIEW_ENTITY_TYPES.SOURCE) {
      const { namespace, name, kind } = entityId as WorkloadId;
      const metric = metrics?.getOverviewMetrics.sources.find((m) => m.kind === kind && m.name === name && m.namespace === namespace);

      const topLimit = -nodeHeight / 2 + framePadding;
      const bottomLimit = Math.floor(containerHeight / nodeHeight) * nodeHeight - (nodeHeight / 2 + framePadding);

      if (position.y >= topLimit && position.y <= bottomLimit) {
        edges.push(
          createEdge(`${nodeId}-to-${actionNodeId}`, {
            animated: false,
            isMultiTarget: false,
            label: formatBytes(metric?.throughput),
            isError: status === STATUSES.UNHEALTHY,
          }),
        );
      }
    }

    if (nodeType === NODE_TYPES.BASE && entityType === OVERVIEW_ENTITY_TYPES.DESTINATION) {
      const metric = metrics?.getOverviewMetrics.destinations.find((m) => m.id === entityId);

      edges.push(
        createEdge(`${actionNodeId}-to-${nodeId}`, {
          animated: false,
          isMultiTarget: true,
          label: formatBytes(metric?.throughput),
          isError: status === STATUSES.UNHEALTHY,
        }),
      );
    }
  });

  return edges;
};
