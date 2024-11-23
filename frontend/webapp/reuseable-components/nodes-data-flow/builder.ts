import theme from '@/styles/theme';
import { Node, Edge } from 'react-flow-renderer';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';
import { extractMonitors, formatBytes, getActionIcon, getEntityIcon, getEntityLabel, getHealthStatus, getRuleIcon, getValueForRange } from '@/utils';
import {
  OVERVIEW_ENTITY_TYPES,
  OVERVIEW_NODE_TYPES,
  STATUSES,
  type InstrumentationRuleSpec,
  type ActualDestination,
  type K8sActualSource,
  type OverviewMetricsResponse,
  ActionDataParsed,
} from '@/types';

const createNode = (nodeId: string, nodeType: string, x: number, y: number, data: Record<string, any>, style?: React.CSSProperties): Node => {
  // const [columnType] = nodeId.split('-');

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
  },
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
  containerWidth,
  containerHeight,
  nodeWidth,
  nodeHeight,
}: {
  rules: InstrumentationRuleSpec[];
  sources: K8sActualSource[];
  actions: ActionDataParsed[];
  destinations: ActualDestination[];
  metrics?: OverviewMetricsResponse;
  containerWidth: number;
  containerHeight: number;
  nodeWidth: number;
  nodeHeight: number;
}) => {
  const nodes: Node[] = [];
  const edges: Edge[] = [];

  if (!containerWidth) {
    return {
      nodes: [],
      edges: [],
    };
  }

  const startX = 24;
  const endX = (containerWidth <= 1500 ? 1500 : containerWidth) - nodeWidth - 40 - startX;
  const getY = (idx?: number) => nodeHeight * ((idx || 0) + 1);

  const postions = {
    rules: {
      x: startX,
      y: getY,
    },
    sources: {
      x: getValueForRange(containerWidth, [
        [0, 1600, endX / 3.5],
        [1600, null, endX / 4],
      ]),
      y: getY,
    },
    actions: {
      x: getValueForRange(containerWidth, [
        [0, 1600, endX / 1.55],
        [1600, null, endX / 1.6],
      ]),
      y: getY,
    },
    destinations: {
      x: endX,
      y: getY,
    },
  };

  const tempNodes = {
    rules: [
      createNode('rule-header', 'header', postions['rules']['x'], 0, {
        icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.RULE),
        title: 'Instrumentation Rules',
        tagValue: rules.length,
      }),
    ],
    sources: [
      createNode('source-header', 'header', postions['sources']['x'], 0, {
        icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.SOURCE),
        title: 'Sources',
        tagValue: sources.length,
      }),
    ],
    actions: [
      createNode('action-header', 'header', postions['actions']['x'] - (!!actions.length ? 15 : 0), 0, {
        icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.ACTION),
        title: 'Actions',
        tagValue: actions.length,
      }),
    ],
    destinations: [
      createNode('destination-header', 'header', postions['destinations']['x'], 0, {
        icon: getEntityIcon(OVERVIEW_ENTITY_TYPES.DESTINATION),
        title: 'Destinations',
        tagValue: destinations.length,
      }),
    ],
  };

  // Build Rules Nodes
  if (!rules.length) {
    tempNodes['rules'].push(
      createNode('rule-0', 'add', postions['rules']['x'], postions['rules']['y'](), {
        type: OVERVIEW_NODE_TYPES.ADD_RULE,
        status: STATUSES.HEALTHY,
        title: 'ADD RULE',
        subTitle: 'Add first rule to modify the OpenTelemetry data',
      }),
    );
  } else {
    rules.forEach((rule, idx) => {
      tempNodes['rules'].push(
        createNode(`rule-${idx}`, 'base', postions['rules']['x'], postions['rules']['y'](idx), {
          id: rule.ruleId,
          type: OVERVIEW_ENTITY_TYPES.RULE,
          status: STATUSES.HEALTHY,
          title: getEntityLabel(rule, OVERVIEW_ENTITY_TYPES.RULE, { prioritizeDisplayName: true }),
          subTitle: rule.type,
          imageUri: getRuleIcon(rule.type),
          isActive: !rule.disabled,
          raw: rule,
        }),
      );
    });
  }

  // Build Source Nodes
  if (!sources.length) {
    tempNodes['sources'].push(
      createNode('source-0', 'add', postions['sources']['x'], postions['rules']['y'](), {
        type: OVERVIEW_NODE_TYPES.ADD_SOURCE,
        status: STATUSES.HEALTHY,
        title: 'ADD SOURCE',
        subTitle: 'Add first source to collect OpenTelemetry data',
      }),
    );
  } else {
    sources.forEach((source, idx) => {
      const metric = metrics?.getOverviewMetrics.sources.find(({ kind, name, namespace }) => kind === source.kind && name === source.name && namespace === source.namespace);

      tempNodes['sources'].push(
        createNode(`source-${idx}`, 'base', postions['sources']['x'], postions['rules']['y'](idx), {
          id: {
            kind: source.kind,
            name: source.name,
            namespace: source.namespace,
          },
          type: OVERVIEW_ENTITY_TYPES.SOURCE,
          status: getHealthStatus(source),
          title: getEntityLabel(source, OVERVIEW_ENTITY_TYPES.SOURCE, { extended: true }),
          subTitle: source.kind,
          imageUri: getMainContainerLanguageLogo(source),
          metric,
          raw: source,
        }),
      );
    });
  }

  // Build Action Nodes
  if (!actions.length) {
    tempNodes['actions'].push(
      createNode('action-0', 'add', postions['actions']['x'], postions['rules']['y'](), {
        type: OVERVIEW_NODE_TYPES.ADD_ACTION,
        status: STATUSES.HEALTHY,
        title: 'ADD ACTION',
        subTitle: 'Add first action to modify the OpenTelemetry data',
      }),
    );
  } else {
    actions.forEach((action, idx) => {
      tempNodes['actions'].push(
        createNode(`action-${idx}`, 'base', postions['actions']['x'], postions['rules']['y'](idx), {
          id: action.id,
          type: OVERVIEW_ENTITY_TYPES.ACTION,
          status: STATUSES.HEALTHY,
          title: getEntityLabel(action, OVERVIEW_ENTITY_TYPES.ACTION, { prioritizeDisplayName: true }),
          subTitle: action.type,
          imageUri: getActionIcon(action.type),
          monitors: action.spec.signals,
          isActive: !action.spec.disabled,
          raw: action,
        }),
      );
    });

    // Create group
    const padding = 15;
    const widthMultiplier = 4.5;
    const heightMultiplier = 1.5;

    tempNodes['actions'].push(
      createNode(
        'action-group',
        'group',
        postions['actions']['x'] - padding,
        postions['rules']['y']() - padding,
        {},
        {
          width: nodeWidth + padding * widthMultiplier,
          height: nodeHeight * actions.length + padding * heightMultiplier,
          background: 'transparent',
          border: `1px dashed ${theme.colors.border}`,
          borderRadius: 24,
          zIndex: -1,
        },
      ),
    );
  }

  // Build Destination Nodes
  if (!destinations.length) {
    tempNodes['destinations'].push(
      createNode('destination-0', 'add', postions['destinations']['x'], postions['rules']['y'](), {
        type: OVERVIEW_NODE_TYPES.ADD_DESTIONATION,
        status: STATUSES.HEALTHY,
        title: 'ADD DESTIONATION',
        subTitle: 'Add first destination to monitor OpenTelemetry data',
      }),
    );
  } else {
    destinations.forEach((destination, idx) => {
      const metric = metrics?.getOverviewMetrics.destinations.find(({ id }) => id === destination.id);

      tempNodes['destinations'].push(
        createNode(`destination-${idx}`, 'base', postions['destinations']['x'], postions['rules']['y'](idx), {
          id: destination.id,
          type: OVERVIEW_ENTITY_TYPES.DESTINATION,
          status: getHealthStatus(destination),
          title: getEntityLabel(destination, OVERVIEW_ENTITY_TYPES.DESTINATION, { prioritizeDisplayName: true }),
          subTitle: destination.destinationType.displayName,
          imageUri: destination.destinationType.imageUrl,
          monitors: extractMonitors(destination.exportedSignals),
          metric,
          raw: destination,
        }),
      );
    });
  }

  // Connect sources to actions
  if (!!sources.length) {
    tempNodes['sources'].forEach((node, idx) => {
      if (idx > 0) {
        const sourceIndex = idx - 1;
        const actionIndex = actions.length ? 'group' : 0;

        edges.push(
          createEdge(`source-${sourceIndex}-to-action-${actionIndex}`, {
            animated: false,
            isMultiTarget: false,
            label: formatBytes(node.data.metric?.throughput),
            isError: node.data.status === STATUSES.UNHEALTHY,
          }),
        );
      }
    });
  }

  // Connect actions to actions
  if (!!actions.length) {
    actions.forEach((_, sourceActionIndex) => {
      if (sourceActionIndex < actions.length - 1) {
        const targetActionIndex = sourceActionIndex + 1;
        edges.push(createEdge(`action-${sourceActionIndex}-to-action-${targetActionIndex}`));
      }
    });
  }

  // Connect actions to destinations
  if (!!destinations.length) {
    tempNodes['destinations'].forEach((node, idx) => {
      if (idx > 0) {
        const destinationIndex = idx - 1;
        const actionIndex = actions.length ? 'group' : 0;

        edges.push(
          createEdge(`action-${actionIndex}-to-destination-${destinationIndex}`, {
            animated: false,
            isMultiTarget: true,
            label: formatBytes(node.data.metric?.throughput),
            isError: node.data.status === STATUSES.UNHEALTHY,
          }),
        );
      }
    });
  }

  tempNodes['rules'].push(
    createNode(
      'hidden',
      'default',
      postions['rules']['x'],
      containerHeight,
      {},
      {
        width: 1,
        height: 1,
        opacity: 0,
        pointerEvents: 'none',
      },
    ),
  );

  Object.values(tempNodes).forEach((arr) => nodes.push(...arr));

  return { nodes, edges };
};
