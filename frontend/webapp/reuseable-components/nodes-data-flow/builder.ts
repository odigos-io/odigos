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
  sources,
  actions,
  destinations,
  columnWidth,
  containerWidth,
}: {
  sources: K8sActualSource[];
  actions: ActionData[];
  destinations: ActualDestination[];
  columnWidth: number;
  containerWidth: number;
}) => {
  // Calculate x positions for each column
  const leftColumnX = 0;
  const rightColumnX = containerWidth - columnWidth;
  const centerColumnX = (containerWidth - columnWidth) / 2;

  // Build Source Nodes
  const sourcesNode: Node[] = [
    createNode('header-source', 'header', leftColumnX, 0, {
      icon: `${HEADER_ICON_PATH}sources.svg`,
      title: 'Sources',
      tagValue: sources.length,
    }),
    ...(!sources.length
      ? [
          createNode(`source-0`, 'add', leftColumnX, NODE_HEIGHT, {
            type: OVERVIEW_NODE_TYPES.ADD_SOURCE,
            title: 'ADD SOURCE',
            subTitle: 'Add first source to collect OpenTelemetry data',
            imageUri: '',
            status: STATUSES.HEALTHY,
          }),
        ]
      : sources.map((source, index) =>
          createNode(`source-${index}`, 'base', leftColumnX, NODE_HEIGHT * (index + 1), {
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

  // Build Destination Nodes
  const destinationNode: Node[] = [
    createNode('header-destination', 'header', rightColumnX, 0, {
      icon: `${HEADER_ICON_PATH}destinations.svg`,
      title: 'Destinations',
      tagValue: destinations.length,
    }),
    ...(!destinations.length
      ? [
          createNode(`destination-0`, 'add', rightColumnX, NODE_HEIGHT, {
            type: OVERVIEW_NODE_TYPES.ADD_DESTIONATION,
            title: 'ADD DESTIONATION',
            subTitle: 'Add first destination to monitor OpenTelemetry data',
            imageUri: '',
            status: STATUSES.HEALTHY,
          }),
        ]
      : destinations.map((destination, index) =>
          createNode(`destination-${index}`, 'base', rightColumnX, NODE_HEIGHT * (index + 1), {
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

  // Build Action Nodes
  const actionsNode: Node[] = [
    createNode('header-action', 'header', centerColumnX, 0, {
      icon: `${HEADER_ICON_PATH}actions.svg`,
      title: 'Actions',
      tagValue: actions.length,
    }),
    ...(!actions.length
      ? [
          createNode(`action-0`, 'add', centerColumnX, NODE_HEIGHT, {
            type: OVERVIEW_NODE_TYPES.ADD_ACTION,
            title: 'ADD ACTION',
            subTitle: 'Add first action to modify the OpenTelemetry data',
            imageUri: '',
            status: STATUSES.HEALTHY,
          }),
        ]
      : actions.map((action, index) => {
          const actionSpec: ActionItem = typeof action.spec === 'string' ? JSON.parse(action.spec) : (action.spec as ActionItem);

          return createNode(`action-${index}`, 'base', centerColumnX, NODE_HEIGHT * (index + 1), {
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

  // Combine all nodes
  const nodes = [...sourcesNode, ...destinationNode, ...actionsNode];

  // Build edges - connecting sources to actions, and actions to destinations
  const edges: Edge[] = [];

  // Connect sources to actions
  const sourceToActionEdges: Edge[] = sources.map((_, sourceIndex) => {
    const actionIndex = actionsNode.length === 2 ? 0 : sourceIndex % actions.length;
    return createEdge(`source-${sourceIndex}-to-action-${actionIndex}`, `source-${sourceIndex}`, `action-${actionIndex}`, false);
  });
  // Connect actions to destinations
  const actionToDestinationEdges: Edge[] = actions.flatMap((_, actionIndex) => {
    return destinations.map((_, destinationIndex) =>
      createEdge(`action-${actionIndex}-to-destination-${destinationIndex}`, `action-${actionIndex}`, `destination-${destinationIndex}`)
    );
  });

  if (actions.length === 0) {
    for (let i = 0; i < destinations.length; i++) {
      actionToDestinationEdges.push(createEdge(`action-0-to-destination-${i}`, `action-0`, `destination-${i}`, false));
    }
  }

  // Combine all edges
  edges.push(...sourceToActionEdges, ...actionToDestinationEdges);

  return { nodes, edges };
};
