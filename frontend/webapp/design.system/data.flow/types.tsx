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
  animated?: boolean;
  style?: { stroke: string };
  data: any;
}

export interface IDataFlow {
  nodes: DataFlowNode[];
  edges: DataFlowEdge[];
}
