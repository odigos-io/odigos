import theme from '@/styles/theme';
import { Node, Edge } from 'react-flow-renderer';
import {
  extractMonitors,
  formatBytes,
  getActionIcon,
  getRuleIcon,
  getValueForRange,
} from '@/utils';
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

const HEADER_ICON_PATH = '/icons/overview/';

const getHealthStatus = (item: K8sActualSource | ActualDestination) => {
  const conditions =
    (item as K8sActualSource)?.instrumentedApplicationDetails?.conditions ||
    (item as ActualDestination)?.conditions;
  const isUnhealthy =
    !conditions?.length ||
    !!conditions.find(({ status }) => status === 'False');

  return isUnhealthy ? STATUSES.UNHEALTHY : STATUSES.HEALTHY;
};

const createNode = (
  nodeId: string,
  nodeType: string,
  x: number,
  y: number,
  data: Record<string, any>,
  style?: React.CSSProperties
): Node => {
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
  containerWidth,
  nodeWidth,
  nodeHeight,
}: {
  rules: InstrumentationRuleSpec[];
  sources: K8sActualSource[];
  actions: ActionData[];
  destinations: ActualDestination[];
  metrics?: OverviewMetricsResponse;
  containerWidth: number;
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

  // Calculate positions for each node
  const startX = 0;
  const endX = (containerWidth <= 1500 ? 1500 : containerWidth) - nodeWidth;
  const postions = {
    rules: {
      x: startX,
      y: (idx?: number) => nodeHeight * ((idx || 0) + 1),
    },
    sources: {
      x: getValueForRange(containerWidth, [
        [0, 1500, endX / 3.5],
        [1500, 1600, endX / 4],
        [1600, null, endX / 4.5],
      ]),
      y: (idx?: number) => nodeHeight * ((idx || 0) + 1),
    },
    actions: {
      x: getValueForRange(containerWidth, [
        [0, 1500, endX / 1.55],
        [1500, 1600, endX / 1.6],
        [1600, null, endX / 1.65],
      ]),
      y: (idx?: number) => nodeHeight * ((idx || 0) + 1),
    },
    destinations: {
      x: endX,
      y: (idx?: number) => nodeHeight * ((idx || 0) + 1),
    },
  };

  const tempNodes = {
    rules: [
      createNode('rule-header', 'header', postions['rules']['x'], 0, {
        icon: `${HEADER_ICON_PATH}rules.svg`,
        title: 'Instrumentation Rules',
        tagValue: rules.length,
      }),
    ],
    sources: [
      createNode('source-header', 'header', postions['sources']['x'], 0, {
        icon: `${HEADER_ICON_PATH}sources.svg`,
        title: 'Sources',
        tagValue: sources.length,
      }),
    ],
    actions: [
      createNode('action-header', 'header', postions['actions']['x'], 0, {
        icon: `${HEADER_ICON_PATH}actions.svg`,
        title: 'Actions',
        tagValue: actions.length,
      }),
    ],
    destinations: [
      createNode(
        'destination-header',
        'header',
        postions['destinations']['x'],
        0,
        {
          icon: `${HEADER_ICON_PATH}destinations.svg`,
          title: 'Destinations',
          tagValue: destinations.length,
        }
      ),
    ],
  };

  // Build Rules Nodes
  if (!rules.length) {
    tempNodes['rules'].push(
      createNode(
        'rule-0',
        'add',
        postions['rules']['x'],
        postions['rules']['y'](),
        {
          type: OVERVIEW_NODE_TYPES.ADD_RULE,
          status: STATUSES.HEALTHY,
          title: 'ADD RULE',
          subTitle: 'Add first rule to modify the OpenTelemetry data',
        }
      )
    );
  } else {
    rules.forEach((rule, idx) => {
      tempNodes['rules'].push(
        createNode(
          `rule-${idx}`,
          'base',
          postions['rules']['x'],
          postions['rules']['y'](idx),
          {
            id: rule.ruleId,
            type: OVERVIEW_ENTITY_TYPES.RULE,
            status: STATUSES.HEALTHY,
            title: rule.ruleName || rule.type,
            subTitle: rule.type,
            imageUri: getRuleIcon(rule.type),
            isActive: !rule.disabled,
          }
        )
      );
    });
  }

  // Build Source Nodes
  if (!sources.length) {
    tempNodes['sources'].push(
      createNode(
        'source-0',
        'add',
        postions['sources']['x'],
        postions['rules']['y'](),
        {
          type: OVERVIEW_NODE_TYPES.ADD_SOURCE,
          status: STATUSES.HEALTHY,
          title: 'ADD SOURCE',
          subTitle: 'Add first source to collect OpenTelemetry data',
        }
      )
    );
  } else {
    sources.forEach((source, idx) => {
      const metric = metrics?.getOverviewMetrics.sources.find(
        ({ kind, name, namespace }) =>
          kind === source.kind &&
          name === source.name &&
          namespace === source.namespace
      );

      tempNodes['sources'].push(
        createNode(
          `source-${idx}`,
          'base',
          postions['sources']['x'],
          postions['rules']['y'](idx),
          {
            id: {
              kind: source.kind,
              name: source.name,
              namespace: source.namespace,
            },
            type: OVERVIEW_ENTITY_TYPES.SOURCE,
            status: getHealthStatus(source),
            title:
              source.name +
              (source.reportedName ? ` (${source.reportedName})` : ''),
            subTitle: source.kind,
            imageUri: getMainContainerLanguageLogo(source),
            metric,
          }
        )
      );
    });
  }

  // Build Action Nodes
  if (!actions.length) {
    tempNodes['actions'].push(
      createNode(
        'action-0',
        'add',
        postions['actions']['x'],
        postions['rules']['y'](),
        {
          type: OVERVIEW_NODE_TYPES.ADD_ACTION,
          status: STATUSES.HEALTHY,
          title: 'ADD ACTION',
          subTitle: 'Add first action to modify the OpenTelemetry data',
        }
      )
    );
  } else {
    actions.forEach((action, idx) => {
      const spec: ActionItem =
        typeof action.spec === 'string'
          ? JSON.parse(action.spec)
          : (action.spec as ActionItem);

      tempNodes['actions'].push(
        createNode(
          `action-${idx}`,
          'base',
          postions['actions']['x'],
          postions['rules']['y'](idx),
          {
            id: action.id,
            type: OVERVIEW_ENTITY_TYPES.ACTION,
            status: STATUSES.HEALTHY,
            title: spec.actionName || action.type,
            subTitle: action.type,
            imageUri: getActionIcon(action.type),
            monitors: spec.signals,
            isActive: !spec.disabled,
          }
        )
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
        }
      )
    );
  }

  // Build Destination Nodes
  if (!destinations.length) {
    tempNodes['destinations'].push(
      createNode(
        'destination-0',
        'add',
        postions['destinations']['x'],
        postions['rules']['y'](),
        {
          type: OVERVIEW_NODE_TYPES.ADD_DESTIONATION,
          status: STATUSES.HEALTHY,
          title: 'ADD DESTIONATION',
          subTitle: 'Add first destination to monitor OpenTelemetry data',
        }
      )
    );
  } else {
    destinations.forEach((destination, idx) => {
      const metric = metrics?.getOverviewMetrics.destinations.find(
        ({ id }) => id === destination.id
      );

      tempNodes['destinations'].push(
        createNode(
          `destination-${idx}`,
          'base',
          postions['destinations']['x'],
          postions['rules']['y'](idx),
          {
            id: destination.id,
            type: OVERVIEW_ENTITY_TYPES.DESTINATION,
            status: getHealthStatus(destination),
            title: destination.name || destination.destinationType.displayName,
            subTitle: destination.destinationType.displayName,
            imageUri: destination.destinationType.imageUrl,
            monitors: extractMonitors(destination.exportedSignals),
            metric,
          }
        )
      );
    });
  }

  // Connect sources to actions
  if (!sources.length) {
    edges.push(createEdge('source-0-to-action-0'));
  } else {
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
          })
        );
      }
    });
  }

  // Connect actions to actions
  if (!!actions.length) {
    actions.forEach((_, sourceActionIndex) => {
      const targetActionIndex = sourceActionIndex + 1;
      edges.push(
        createEdge(`action-${sourceActionIndex}-to-action-${targetActionIndex}`)
      );
    });
  }

  // Connect actions to destinations
  if (!destinations.length) {
    edges.push(createEdge('action-0-to-destination-0'));
  } else {
    tempNodes['destinations'].forEach((node, idx) => {
      if (idx > 0) {
        const destinationIndex = idx - 1;
        const actionIndex = actions.length ? 'group' : 0;

        edges.push(
          createEdge(
            `action-${actionIndex}-to-destination-${destinationIndex}`,
            {
              animated: false,
              isMultiTarget: true,
              label: formatBytes(node.data.metric?.throughput),
              isError: node.data.status === STATUSES.UNHEALTHY,
            }
          )
        );
      }
    });
  }

  Object.values(tempNodes).forEach((arr) => nodes.push(...arr));

  return { nodes, edges };
};
