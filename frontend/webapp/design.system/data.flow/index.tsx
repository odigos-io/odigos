"use client";
import React, { use, useCallback, useEffect } from "react";
import ReactFlow, {
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
  Controls,
  useReactFlow,
  ReactFlowProvider,
} from "reactflow";
import CustomNode from "./CustomNode";
import NamespaceNode from "./namespace.node";
import DestinationNode from "./destination.node";
import "reactflow/dist/style.css";

const initialNodes = [
  {
    id: "1",
    type: "namespace",
    data: null,
    position: { x: 100, y: 100 },
  },
  {
    id: "4",
    type: "namespace",
    position: { x: 100, y: 200 },
    data: null,
  },
  {
    id: "5",
    type: "namespace",
    position: { x: 100, y: 300 },
    data: null,
  },
  {
    id: "7",
    type: "namespace",
    position: { x: 100, y: 400 },
    data: null,
  },
  {
    id: "8",
    type: "namespace",
    position: { x: 100, y: 500 },
    data: null,
  },
  {
    id: "2",
    type: "custom",
    data: null,

    position: { x: 400, y: 300 },
  },
  {
    id: "3",
    type: "destination",
    position: { x: 530, y: 100 },
    data: null,
  },
  {
    id: "6",
    type: "destination",
    position: { x: 530, y: 200 },
    data: null,
  },
];

const initialEdges = [
  {
    id: "e1-2",
    source: "1",
    target: "2",
    style: { stroke: "#96f3ff8e" },
    data: null,
  },
  {
    id: "e1-4",
    source: "4",
    target: "2",
    style: { stroke: "#96f3ff8e" },
    data: null,
  },
  {
    id: "e1-5",
    source: "5",
    target: "2",
    style: { stroke: "#96f3ff8e" },
    data: null,
  },
  {
    id: "e1-7",
    source: "7",
    target: "2",
    style: { stroke: "#96f3ff8e" },
    data: null,
  },
  {
    id: "e1-8",
    source: "8",
    target: "2",
    style: { stroke: "#96f3ff8e" },
    data: null,
  },
  {
    id: "e2-3",
    source: "2",
    target: "3",
    animated: true,
    style: { stroke: "#96f3ff8e" },
    data: null,
  },
  {
    id: "e2-1",
    source: "2",
    target: "3",
    animated: true,
    style: { stroke: "#96f3ff8e" },
    data: null,
  },
  {
    id: "e2-1",
    source: "2",
    target: "6",
    animated: true,
    style: { stroke: "#96f3ff8e" },
    data: null,
  },
];

const nodeTypes = {
  custom: CustomNode,
  namespace: NamespaceNode,
  destination: DestinationNode,
};

function KeyvalDataFlow() {
  const [nodes, setNodes, onNodesChange] = useNodesState<any>(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const { zoomTo, fitView } = useReactFlow();

  useEffect(() => {
    setTimeout(() => {
      fitView();
      zoomTo(1);
    }, 0);
  }, [fitView]);

  const onConnect = useCallback(
    (params) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  return (
    <div style={{ width: "100%", height: "100%" }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        nodesDraggable={false}
      >
        <Controls />
        <Background gap={12} size={1} style={{ backgroundColor: "#132330" }} />
      </ReactFlow>
    </div>
  );
}

export function KeyvalFlow(props) {
  return (
    <ReactFlowProvider>
      <KeyvalDataFlow {...props} />
    </ReactFlowProvider>
  );
}
