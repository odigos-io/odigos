interface SourceData {
  namespace: string;
}

interface DataFlowEdge {
  id: string;
}

interface DataFlowNode {
  id: string;
  type: string;
  data: SourceData | null;
  position: { x: number; y: number };
}

interface GroupedSource {
  name: string;
  totalAppsInstrumented: number;
}

export function groupSourcesNamespace(
  sources: SourceData[] | null
): GroupedSource[] {
  if (!sources) return [];
  const groupedSources: { [key: string]: GroupedSource } = sources.reduce(
    (result, item) => {
      const propertyValue = item?.namespace;
      if (!result[propertyValue]) {
        result[propertyValue] = {
          name: propertyValue,
          totalAppsInstrumented: 0,
        };
      }
      result[propertyValue].totalAppsInstrumented += 1;
      return result;
    },
    {}
  );

  return Object.values(groupedSources);
}

export function getNodes(
  height: number,
  nodeData: any,
  type: string,
  listItemHeight: number,
  xPosition: number,
  addCenterNode: boolean = false
): DataFlowNode[] {
  if (!nodeData || isNaN(height)) return [];
  const nodes: DataFlowNode[] = [];
  const totalListItemsHeight = nodeData.length * listItemHeight;

  let topPosition = (height - totalListItemsHeight) / 2;
  const centerIndex = Math.floor(nodeData.length / 2);
  nodeData.forEach((data, index) => {
    const y = topPosition;
    nodes.push({
      id: `${type}-${index}`,
      type,
      data,
      position: { x: xPosition, y },
    });
    if (index === centerIndex && addCenterNode) {
      nodes.push({
        id: 'centerNode',
        type: 'custom',
        data: null,
        position: { x: 400, y },
      });
    }
    topPosition += listItemHeight;
  });
  return nodes;
}

export function getEdges(
  destinations: DataFlowEdge[],
  sources: DataFlowEdge[]
) {
  return [
    ...destinations.flatMap((node, index) => ({
      id: `edges-${node.id}`,
      source: 'centerNode',
      target: `destination-${index}`,
      animated: true,
      style: { stroke: '#96f3ff8e' },
      data: null,
    })),
    ...sources.flatMap((node, index) => ({
      id: `edges-${node.id}`,
      source: `namespace-${index}`,
      target: 'centerNode',
      animated: true,
      style: { stroke: '#96f3ff8e' },
      data: null,
    })),
  ];
}
