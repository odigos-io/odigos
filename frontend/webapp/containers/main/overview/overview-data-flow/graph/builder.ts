import { Node, Edge } from 'react-flow-renderer';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';
import { ActionData, ActualDestination, K8sActualSource } from '@/types';

interface BuildNodesAndEdgesProps {
  sources: K8sActualSource[];
  actions: ActionData[];
  destinations: ActualDestination[];
  columnWidth: number;
  containerWidth: number;
}

export const buildNodesAndEdges = ({
  sources,
  actions,
  destinations,
  columnWidth,
  containerWidth,
}: BuildNodesAndEdgesProps) => {
  const leftColumnX = 0;
  const rightColumnX = containerWidth - columnWidth;
  const centerColumnX = (containerWidth - columnWidth) / 2;

  const nodes: Node[] = [];

  // Source Nodes
  const sourcesNode: Node[] = [
    {
      type: 'header',
      id: 'header-source',
      position: { x: leftColumnX, y: 0 },
      data: {
        icon: '/icons/overview/sources.svg',
        title: 'Sources',
        tagValue: sources.length,
      },
    },
    ...sources.map((source, index) => ({
      id: `source-${index}`,
      type: 'base',
      position: { x: leftColumnX, y: 80 * (index + 1) },
      data: {
        type: 'source',
        title: source.name,
        subTitle: source.kind,
        imageUri: getMainContainerLanguageLogo(source),
        status: 'healthy',
        onClick: () => console.log(source),
      },
    })),
  ];

  // Destination Nodes
  const destinationNode: Node[] = [
    {
      type: 'header',
      id: 'header-destination',
      position: { x: rightColumnX, y: 0 },
      data: {
        icon: '/icons/overview/destinations.svg',
        title: 'Destinations',
        tagValue: destinations.length,
      },
    },
    ...destinations.map((destination, index) => ({
      id: `destination-${index}`,
      type: 'base',
      position: { x: rightColumnX, y: 80 * (index + 1) },
      data: {
        type: 'destination',
        title: destination.destinationType.displayName,
        subTitle: 'Destination',
        imageUri: destination.destinationType.imageUrl,
        status: 'healthy',
        onClick: () => console.log(destination),
      },
    })),
  ];

  // Actions Nodes
  const actionsNode: Node[] = [
    {
      type: 'header',
      id: 'header-action',
      position: { x: centerColumnX, y: 0 },
      data: {
        icon: '/icons/overview/actions.svg',
        title: 'Actions',
        tagValue: actions.length,
      },
    },
    ...actions.map((action, index) => ({
      id: `action-${index}`,
      type: 'base',
      position: { x: centerColumnX, y: 80 * (index + 1) },
      data: {
        type: 'action',
        title: action.type,
        subTitle: 'Action',
        imageUri: '/icons/common/action.svg',
        status: 'healthy',
        onClick: () => console.log(action),
      },
    })),
  ];

  // Combine all nodes
  nodes.push(...sourcesNode, ...destinationNode, ...actionsNode);

  // Edges - Connecting sources to actions and actions to destinations
  const edges: Edge[] = [];

  const sourceToActionEdges: Edge[] = sources.map((_, sourceIndex) => {
    const actionIndex = sourceIndex % actions.length;
    return {
      id: `source-${sourceIndex}-to-action-${actionIndex}`,
      source: `source-${sourceIndex}`,
      target: `action-${actionIndex}`,
      animated: true,
      style: { stroke: '#525252' },
    };
  });

  const actionToDestinationEdges: Edge[] = actions.flatMap((_, actionIndex) => {
    return destinations.map((_, destinationIndex) => ({
      id: `action-${actionIndex}-to-destination-${destinationIndex}`,
      source: `action-${actionIndex}`,
      target: `destination-${destinationIndex}`,
      animated: true,
      style: { stroke: '#525252' },
    }));
  });

  edges.push(...sourceToActionEdges, ...actionToDestinationEdges);

  return { nodes, edges };
};
