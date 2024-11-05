import theme from '@/styles/theme';
import { getActionIcon } from '@/utils';
import { Node, Edge } from 'react-flow-renderer';
import { getRuleIcon } from '@/utils/functions';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';
import {
  OVERVIEW_ENTITY_TYPES,
  OVERVIEW_NODE_TYPES,
  STATUSES,
  type InstrumentationRuleSpec,
  type ActionData,
  type ActionItem,
  type ActualDestination,
  type K8sActualSource,
  type OverviewMetricsResponse,
} from '@/types';

const NODE_HEIGHT = 80;
const HEADER_ICON_PATH = '/icons/overview/';

const extractMonitors = (exportedSignals: Record<string, boolean>) => {
  const filtered = Object.keys(exportedSignals).filter((signal) => exportedSignals[signal] === true);

  return filtered;
};

const getDifference = (containerWidth: number, columnWidth: number) => {
  const minWidth = 1500;
  const diff = (containerWidth <= minWidth ? minWidth : containerWidth) - columnWidth;

  return diff;
};

const getValueForRange = (current: number, matrix: (number | null)[][]) => {
  const found = matrix.find(([val, min, max]) => (min === null || current >= min) && (max === null || current <= max));

  return found?.[0] || 0;
};

const formatBytes = (bytes?: number) => {
  if (!bytes) return '0 KB/s';

  const sizes = ['Bytes', 'KB/s', 'MB/s', 'GB/s', 'TB/s'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const value = bytes / Math.pow(1024, i);

  return `${value.toFixed(2)} ${sizes[i]}`;
};

const getHealthStatus = (item: K8sActualSource | ActualDestination) => {
  const conditions = (item as K8sActualSource)?.instrumentedApplicationDetails?.conditions || (item as ActualDestination)?.conditions;
  const isUnhealthy = !conditions?.length || !!conditions.find(({ status }) => status === 'False');

  return isUnhealthy ? STATUSES.UNHEALTHY : STATUSES.HEALTHY;
};

const createNode = (nodeId: string, nodeType: string, x: number, y: number, data: Record<string, any>, style?: React.CSSProperties): Node => {
  // const [columnType] = id.split('-');

  return {
    id: nodeId,
    type: nodeType,
    data,
    style,
    position: { x, y },
  };
};

const createEdge = (
  edgeId: string,
  params?: {
    label?: string;
    isMultiTarget?: boolean;
    isError?: boolean;
    animated?: boolean;
  }
): Edge => {
  const { label, isMultiTarget, isError, animated } = params || {};
  const [sourceNodeId, targetNodeId] = edgeId.split('-to-');

  return {
    id: edgeId,
    type: !!label ? 'labeled' : 'default',
    source: sourceNodeId,
    target: targetNodeId,
    animated,
    data: { label, isMultiTarget, isError },
    style: { stroke: isError ? theme.colors.dark_red : theme.colors.border },
  };
};

export const buildNodesAndEdges = ({
  rules,
  sources,
  actions,
  destinations,
  metrics,
  columnWidth,
  containerWidth,
}: {
  rules: InstrumentationRuleSpec[];
  sources: K8sActualSource[];
  actions: ActionData[];
  destinations: ActualDestination[];
  metrics?: OverviewMetricsResponse;
  columnWidth: number;
  containerWidth: number;
}) => {
  if (!containerWidth) {
    return {
      nodes: [],
      edges: [],
    };
  }

  // Calculate x positions for each column
  const difference = getDifference(containerWidth, columnWidth);
  const columnPostions = {
    rules: 0,
    sources:
      difference /
      getValueForRange(containerWidth, [
        [3.5, 0, 1500],
        [4, 1500, 1600],
        [4.5, 1600, null],
      ]),
    actions:
      difference /
      getValueForRange(containerWidth, [
        [1.55, 0, 1500],
        [1.6, 1500, 1600],
        [1.65, 1600, null],
      ]),
    destinations: difference,
  };

  // Build Rules Nodes
  const ruleNodes: Node[] = [
    createNode('rule-header', 'header', columnPostions['rules'], 0, {
      icon: `${HEADER_ICON_PATH}rules.svg`,
      title: 'Instrumentation Rules',
      tagValue: rules.length,
    }),
    ...(!rules.length
      ? [
          createNode('rule-0', 'add', columnPostions['rules'], NODE_HEIGHT, {
            type: OVERVIEW_NODE_TYPES.ADD_RULE,
            status: STATUSES.HEALTHY,
            title: 'ADD RULE',
            subTitle: 'Add first rule to modify the OpenTelemetry data',
          }),
        ]
      : rules.map((rule, index) =>
          createNode(`rule-${index}`, 'base', columnPostions['rules'], NODE_HEIGHT * (index + 1), {
            id: rule.ruleId,
            type: OVERVIEW_ENTITY_TYPES.RULE,
            status: STATUSES.HEALTHY,
            title: rule.ruleName || rule.type,
            subTitle: rule.type,
            imageUri: getRuleIcon(rule.type),
            isActive: !rule.disabled,
          })
        )),
  ];

  // Build Source Nodes
  const sourceNodes: Node[] = [
    createNode('source-header', 'header', columnPostions['sources'], 0, {
      icon: `${HEADER_ICON_PATH}sources.svg`,
      title: 'Sources',
      tagValue: sources.length,
    }),
    ...(!sources.length
      ? [
          createNode('source-0', 'add', columnPostions['sources'], NODE_HEIGHT, {
            type: OVERVIEW_NODE_TYPES.ADD_SOURCE,
            status: STATUSES.HEALTHY,
            title: 'ADD SOURCE',
            subTitle: 'Add first source to collect OpenTelemetry data',
          }),
        ]
      : sources.map((source, index) => {
          const metric = metrics?.sources.find(
            ({ kind, name, namespace }) => kind === source.kind && name === source.name && namespace === source.namespace
          );

          return createNode(`source-${index}`, 'base', columnPostions['sources'], NODE_HEIGHT * (index + 1), {
            id: { kind: source.kind, name: source.name, namespace: source.namespace },
            type: OVERVIEW_ENTITY_TYPES.SOURCE,
            status: getHealthStatus(source),
            title: source.name + (source.reportedName ? ` (${source.reportedName})` : ''),
            subTitle: source.kind,
            imageUri: getMainContainerLanguageLogo(source),
            metric,
          });
        })),
  ];

  // Build Action Nodes
  const actionNodes: Node[] = [
    createNode('action-header', 'header', columnPostions['actions'], 0, {
      icon: `${HEADER_ICON_PATH}actions.svg`,
      title: 'Actions',
      tagValue: actions.length,
    }),
    ...(!actions.length
      ? [
          createNode('action-0', 'add', columnPostions['actions'], NODE_HEIGHT, {
            type: OVERVIEW_NODE_TYPES.ADD_ACTION,
            status: STATUSES.HEALTHY,
            title: 'ADD ACTION',
            subTitle: 'Add first action to modify the OpenTelemetry data',
          }),
        ]
      : actions.map((action, index) => {
          const actionSpec: ActionItem = typeof action.spec === 'string' ? JSON.parse(action.spec) : (action.spec as ActionItem);

          return createNode(`action-${index}`, 'base', columnPostions['actions'], NODE_HEIGHT * (index + 1), {
            id: action.id,
            type: OVERVIEW_ENTITY_TYPES.ACTION,
            status: STATUSES.HEALTHY,
            title: actionSpec.actionName || action.type,
            subTitle: action.type,
            imageUri: getActionIcon(action.type),
            monitors: actionSpec.signals,
            isActive: !actionSpec.disabled,
          });
        })),
  ];

  // Create group for actions
  if (actions.length) {
    const padding = 15;
    const getDifference = (x: number) => {
      const a = 23.24; // coefficient
      const b = -0.589; // exponent
      return a * Math.pow(x, b);
    };

    actionNodes.push(
      createNode(
        'action-group',
        'group',
        columnPostions['actions'] - padding,
        NODE_HEIGHT - padding,
        {},
        {
          width: columnWidth + padding * getDifference(padding),
          height: NODE_HEIGHT * actions.length + padding,
          background: 'transparent',
          border: `1px dashed ${theme.colors.border}`,
          borderRadius: 24,
          zIndex: -1,
        }
      )
    );
  }

  // Build Destination Nodes
  const destinationNodes: Node[] = [
    createNode('destination-header', 'header', columnPostions['destinations'], 0, {
      icon: `${HEADER_ICON_PATH}destinations.svg`,
      title: 'Destinations',
      tagValue: destinations.length,
    }),
    ...(!destinations.length
      ? [
          createNode('destination-0', 'add', columnPostions['destinations'], NODE_HEIGHT, {
            type: OVERVIEW_NODE_TYPES.ADD_DESTIONATION,
            status: STATUSES.HEALTHY,
            title: 'ADD DESTIONATION',
            subTitle: 'Add first destination to monitor OpenTelemetry data',
          }),
        ]
      : destinations.map((destination, index) => {
          const metric = metrics?.destinations.find(({ id }) => id === destination.id);

          return createNode(`destination-${index}`, 'base', columnPostions['destinations'], NODE_HEIGHT * (index + 1), {
            id: destination.id,
            type: OVERVIEW_ENTITY_TYPES.DESTINATION,
            status: getHealthStatus(destination),
            title: destination.name,
            subTitle: destination.destinationType.displayName,
            imageUri: destination.destinationType.imageUrl,
            monitors: extractMonitors(destination.exportedSignals),
            metric,
          });
        })),
  ];

  // Combine all nodes
  const nodes = [...ruleNodes, ...sourceNodes, ...actionNodes, ...destinationNodes];

  // Build edges - connecting sources to actions, and actions to destinations
  const edges: Edge[] = [];

  // Connect sources to actions
  if (!sources.length) {
    edges.push(createEdge('source-0-to-action-0'));
  } else {
    sourceNodes.forEach((node, idx) => {
      if (idx > 0) {
        const sourceIndex = idx - 1;
        const actionIndex = actions.length ? 'group' : 0;

        edges.push(
          createEdge(`source-${sourceIndex}-to-action-${actionIndex}`, {
            isMultiTarget: false,
            label: formatBytes(node.data.metric?.throughput),
            isError: node.data.status === STATUSES.UNHEALTHY,
          })
        );
      }
    });
  }

  // Connect actions to actions
  if (!!actions.length) {
    actions.forEach((_, sourceActionIndex) => {
      const targetActionIndex = sourceActionIndex + 1;
      edges.push(createEdge(`action-${sourceActionIndex}-to-action-${targetActionIndex}`));
    });
  }

  // Connect actions to destinations
  if (!destinations.length) {
    edges.push(createEdge('action-0-to-destination-0'));
  } else {
    destinationNodes.forEach((node, idx) => {
      if (idx > 0) {
        const destinationIndex = idx - 1;
        const actionIndex = actions.length ? 'group' : 0;

        edges.push(
          createEdge(`action-${actionIndex}-to-destination-${destinationIndex}`, {
            isMultiTarget: true,
            label: formatBytes(node.data.metric?.throughput),
            isError: node.data.status === STATUSES.UNHEALTHY,
          })
        );
      }
    });
  }

  return { nodes, edges };
};
