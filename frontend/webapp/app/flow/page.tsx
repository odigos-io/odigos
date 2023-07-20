"use client";
import React, { useCallback } from "react";
import ReactFlow, {
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
  Controls,
} from "reactflow";
import CustomNode from "./CustomNode";
import NamespaceNode from "./namespace.node";
import DestinationNode from "./destination.node";
import "reactflow/dist/style.css";
import ConnectionLine from "./connection.line";

const initialNodes = [
  {
    id: "1",
    type: "namespace",
    // data: { label: "Input Node" },
    position: { x: 100, y: 100 },
  },
  {
    id: "4",
    type: "namespace",
    position: { x: 100, y: 200 },
  },
  {
    id: "5",
    type: "namespace",
    position: { x: 100, y: 300 },
  },
  {
    id: "2",
    type: "custom",
    // you can also pass a React component as a label

    position: { x: 400, y: 300 },
  },
  {
    id: "3",
    type: "destination",
    position: { x: 530, y: 100 },
  },
  {
    id: "6",
    type: "destination",
    position: { x: 530, y: 200 },
  },
];

const initialEdges = [
  {
    id: "e1-2",
    source: "1",
    target: "2",
    style: { stroke: "#96f3ff8e" },
  },
  { id: "e1-4", source: "4", target: "2", style: { stroke: "#96f3ff8e" } },
  { id: "e1-5", source: "5", target: "2", style: { stroke: "#96f3ff8e" } },
  {
    id: "e2-3",
    source: "2",
    target: "3",
    animated: true,
    style: { stroke: "#96f3ff8e" },
  },
  {
    id: "e2-1",
    source: "2",
    target: "3",
    animated: true,
    style: { stroke: "#96f3ff8e" },
  },
  {
    id: "e2-1",
    source: "2",
    target: "6",
    animated: true,
    style: { stroke: "#96f3ff8e" },
  },
];

const nodeTypes = {
  custom: CustomNode,
  namespace: NamespaceNode,
  destination: DestinationNode,
};

const edgeTypes = {
  custom: ConnectionLine,
};

export default function App() {
  const [nodes, setNodes, onNodesChange] = useNodesState<any>(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  const onConnect = useCallback(
    (params) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  return (
    <div style={{ width: "100vw", height: "100vh" }}>
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
