import theme from '@/styles/theme';
import { getActionIcon } from '@/utils';
import { Node, Edge } from 'react-flow-renderer';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';
import {
  OVERVIEW_ENTITY_TYPES,
  OVERVIEW_NODE_TYPES,
  STATUSES,
  type ActionData,
  type ActionItem,
  type ActualDestination,
  type K8sActualSource,
} from '@/types';

// Constants
const NODE_HEIGHT = 80;

const STROKE_COLOR = theme.colors.border;
const HEADER_ICON_PATH = '/icons/overview/';

// Helper to create a node
const createNode = (id: string, type: string, x: number, y: number, data: Record<string, any>): Node => ({
  id,
  type,
  position: { x, y },
  data,
});

// Helper to create an edge
const createEdge = (id: string, source: string, target: string, animated = true): Edge => ({
  id,
  source,
  target,
  animated,
  style: { stroke: STROKE_COLOR },
});

// Extract the monitors from exported signals
const extractMonitors = (exportedSignals: Record<string, boolean>) =>
  Object.keys(exportedSignals).filter((signal) => exportedSignals[signal] === true);

export const buildNodesAndEdges = ({
  rules,
  sources,
  actions,
  destinations,
  columnWidth,
  containerWidth,
}: {
  rules: any[];
  sources: K8sActualSource[];
  actions: ActionData[];
  destinations: ActualDestination[];
  columnWidth: number;
  containerWidth: number;
}) => {
  // Calculate x positions for each column
  const columnPostions = {
    rules: 0,
    sources: (containerWidth - columnWidth) / 3,
    actions: (containerWidth - columnWidth) / 1.5,
    destinations: containerWidth - columnWidth,
  };

  // Build Rules Nodes
  const ruleNodes: Node[] = [
    createNode('header-rule', 'header', columnPostions['rules'], 0, {
      icon: `${HEADER_ICON_PATH}rules.svg`,
      title: 'Instrumentation Rules',
      tagValue: rules.length,
    }),
    ...(!rules.length
      ? [
          createNode(`rule-0`, 'add', columnPostions['rules'], NODE_HEIGHT, {
            type: OVERVIEW_NODE_TYPES.ADD_RULE,
            title: 'ADD RULE',
            subTitle: 'Add first rule to modify the OpenTelemetry data',
            imageUri: '',
            status: STATUSES.HEALTHY,
          }),
        ]
      : rules.map((rule, index) =>
          createNode(`rule-${index}`, 'base', columnPostions['rules'], NODE_HEIGHT * (index + 1), {
            id: rule.id,
            type: OVERVIEW_ENTITY_TYPES.RULE,
            status: STATUSES.HEALTHY,
            title: rule.actionName || rule.type,
            subTitle: rule.type,
            imageUri: '',
            isActive: false,
          })
        )),
  ];

  // Build Source Nodes
  const sourceNodes: Node[] = [
    createNode('header-source', 'header', columnPostions['sources'], 0, {
      icon: `${HEADER_ICON_PATH}sources.svg`,
      title: 'Sources',
      tagValue: sources.length,
    }),
    ...(!sources.length
      ? [
          createNode(`source-0`, 'add', columnPostions['sources'], NODE_HEIGHT, {
            type: OVERVIEW_NODE_TYPES.ADD_SOURCE,
            title: 'ADD SOURCE',
            subTitle: 'Add first source to collect OpenTelemetry data',
            imageUri: '',
            status: STATUSES.HEALTHY,
          }),
        ]
      : sources.map((source, index) =>
          createNode(`source-${index}`, 'base', columnPostions['sources'], NODE_HEIGHT * (index + 1), {
            type: OVERVIEW_ENTITY_TYPES.SOURCE,
            title: source.name + (source.reportedName ? ` (${source.reportedName})` : ''),
            subTitle: source.kind,
            imageUri: getMainContainerLanguageLogo(source),
            status: STATUSES.HEALTHY,
            id: {
              kind: source.kind,
              name: source.name,
              namespace: source.namespace,
            },
          })
        )),
  ];

  // Build Action Nodes
  const actionNodes: Node[] = [
    createNode('header-action', 'header', columnPostions['actions'], 0, {
      icon: `${HEADER_ICON_PATH}actions.svg`,
      title: 'Actions',
      tagValue: actions.length,
    }),
    ...(!actions.length
      ? [
          createNode(`action-0`, 'add', columnPostions['actions'], NODE_HEIGHT, {
            type: OVERVIEW_NODE_TYPES.ADD_ACTION,
            title: 'ADD ACTION',
            subTitle: 'Add first action to modify the OpenTelemetry data',
            imageUri: '',
            status: STATUSES.HEALTHY,
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

  // Build Destination Nodes
  const destinationNodes: Node[] = [
    createNode('header-destination', 'header', columnPostions['destinations'], 0, {
      icon: `${HEADER_ICON_PATH}destinations.svg`,
      title: 'Destinations',
      tagValue: destinations.length,
    }),
    ...(!destinations.length
      ? [
          createNode(`destination-0`, 'add', columnPostions['destinations'], NODE_HEIGHT, {
            type: OVERVIEW_NODE_TYPES.ADD_DESTIONATION,
            title: 'ADD DESTIONATION',
            subTitle: 'Add first destination to monitor OpenTelemetry data',
            imageUri: '',
            status: STATUSES.HEALTHY,
          }),
        ]
      : destinations.map((destination, index) =>
          createNode(`destination-${index}`, 'base', columnPostions['destinations'], NODE_HEIGHT * (index + 1), {
            type: OVERVIEW_ENTITY_TYPES.DESTINATION,
            title: destination.name,
            subTitle: destination.destinationType.displayName,
            imageUri: destination.destinationType.imageUrl,
            status: STATUSES.HEALTHY,
            monitors: extractMonitors(destination.exportedSignals),
            id: destination.id,
          })
        )),
  ];

  // Combine all nodes
  const nodes = [...ruleNodes, ...sourceNodes, ...actionNodes, ...destinationNodes];

  // Build edges - connecting sources to actions, and actions to destinations
  const edges: Edge[] = [];

  // Connect sources to actions
  if (!sources.length) {
    edges.push(createEdge('source-0-to-action-0', 'source-0', 'action-0', false));
  } else {
    sources.forEach((_, sourceIndex) => {
      const actionIndex = 0;
      edges.push(createEdge(`source-${sourceIndex}-to-action-${actionIndex}`, `source-${sourceIndex}`, `action-${actionIndex}`, false));
    });
  }

  // Connect actions to destinations
  if (!destinations.length) {
    edges.push(createEdge('action-0-to-destination-0', 'action-0', 'destination-0'));
  } else {
    destinations.forEach((_, destinationIndex) => {
      const actionIndex = !actions.length ? 0 : actions.length - 1;
      edges.push(createEdge(`action-${actionIndex}-to-destination-${destinationIndex}`, `action-${actionIndex}`, `destination-${destinationIndex}`));
    });
  }

  return { nodes, edges };
};
