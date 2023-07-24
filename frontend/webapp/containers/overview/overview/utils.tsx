interface DataFlowNode {
  id: string;
  type: string;
  data: any;
  position: { x: number; y: number };
}

interface DataFlowEdge {
  id: string;
  source: string;
  target: string;
  animated: boolean;
  style: { stroke: string };
  data: any;
}

export function getNodes(
  height: number,
  nodeData: DataFlowNode[],
  type: string,
  listItemHeight: number,
  xPosition: number,
  addCenterNode: boolean = false
) {
  if (!nodeData || isNaN(height)) return [];
  let nodes: any = [];
  const totalListItemsHeight = nodeData?.length * listItemHeight;

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
        id: "1",
        type: "custom",
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
      source: "1",
      target: `destination-${index}`,
      animated: true,
      style: { stroke: "#96f3ff8e" },
      data: null,
    })),
    ...sources.flatMap((node, index) => ({
      id: `edges-${node.id}`,
      source: `namespace-${index}`,
      target: "1",
      animated: true,
      style: { stroke: "#96f3ff8e" },
      data: null,
    })),
  ];
}
