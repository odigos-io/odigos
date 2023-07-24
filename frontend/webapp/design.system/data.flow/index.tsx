"use client";
import React, { useCallback, useEffect, useRef, useState } from "react";
import ReactFlow, {
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
  Controls,
  useReactFlow,
  ReactFlowProvider,
} from "reactflow";
import CustomNode from "./keyval.middleware";
import NamespaceNode from "./namespace.node";
import DestinationNode from "./destination.node";
import "reactflow/dist/style.css";

const initialNodes = [
  {
    id: "1",
    type: "namespace",
    data: null,
    position: { x: 0, y: 100 },
  },
  {
    id: "4",
    type: "namespace",
    position: { x: 0, y: 200 },
    data: null,
  },
  {
    id: "5",
    type: "namespace",
    position: { x: 0, y: 300 },
    data: null,
  },
  {
    id: "7",
    type: "namespace",
    position: { x: 0, y: 400 },
    data: null,
  },
  {
    id: "8",
    type: "namespace",
    position: { x: 0, y: 500 },
    data: null,
  },
  {
    id: "2",
    type: "custom",
    data: null,

    position: { x: 385, y: 300 },
  },
  {
    id: "3",
    type: "destination",
    position: { x: 750, y: -100 },
    data: null,
  },
  {
    id: "6",
    type: "destination",
    position: { x: 750, y: 200 },
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
    animated: true,
  },
  {
    id: "e1-4",
    source: "4",
    target: "2",
    style: { stroke: "#96f3ff8e" },
    data: null,
    animated: true,
  },
  {
    id: "e1-5",
    source: "5",
    target: "2",
    style: { stroke: "#96f3ff8e" },
    data: null,
    animated: true,
  },
  {
    id: "e1-7",
    source: "7",
    target: "2",
    style: { stroke: "#96f3ff8e" },
    data: null,
    animated: true,
  },
  {
    id: "e1-8",
    source: "8",
    target: "2",
    style: { stroke: "#96f3ff8e" },
    data: null,
    animated: true,
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
    id: "e2-01",
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

function KeyvalDataFlow({ sources, destinations }) {
  const [namespaceNodes, setNamespaceNodes] = useState([]);
  const [initialEdges, setInitialEdges] = useState([]);
  const containerRef = useRef(null);

  const { zoomTo, fitView } = useReactFlow();

  useEffect(() => {
    setTimeout(() => {
      fitView();
      zoomTo(1);
    }, 100);
  }, [fitView, namespaceNodes]);

  useEffect(() => {
    destinations && centerDestinationListVertically();
  }, [destinations]);

  function centerDestinationListVertically() {
    const canvasHeight = containerRef.current?.clientHeight;
    const listItemHeight = 120; // Adjust this value to the desired height of each list item
    const totalListItemsHeight = destinations.length * listItemHeight;

    let topPosition = (canvasHeight - totalListItemsHeight) / 2;

    let nodes: any = [
      {
        id: "1",
        type: "custom",
        data: null,

        position: { x: 385, y: 300 },
      },
      ...getDestinationNodes(),
      ...getSourcesNodes(),
    ];

    // destinations.forEach((data, index) => {
    //   const y = topPosition;
    //   nodes.push({
    //     id: `source-${index}`,
    //     type: "destination",
    //     data,
    //     position: { x: 800, y },
    //   });
    //   topPosition += 100;
    // });

    const destinations_edges = nodes.map((node, index) => {
      console.log({ node });
      return {
        id: `edges-${node.id}`,
        source: "1",
        target: `destination-${index}`,
        animated: true,
        style: { stroke: "#96f3ff8e" },
        data: null,
      };
    });

    const sources_edges = nodes.map((node, index) => {
      return {
        id: `edges-${node.id}`,
        source: `source-${index}`,
        target: "1",
        animated: true,
        style: { stroke: "#96f3ff8e" },
        data: null,
      };
    });

    setInitialEdges([...sources_edges, ...destinations_edges]);
    setNamespaceNodes(nodes);
    // setTimeout(() => {
    //   fitView();
    //   zoomTo(1);
    // }, 1000);
  }

  function getDestinationNodes() {
    let nodes: any = [];
    const canvasHeight = containerRef.current?.clientHeight;
    const listItemHeight = 120; // Adjust this value to the desired height of each list item
    const totalListItemsHeight = destinations.length * listItemHeight;

    let topPosition = (canvasHeight - totalListItemsHeight) / 2;

    destinations.forEach((data, index) => {
      const y = topPosition;
      nodes.push({
        id: `destination-${index}`,
        type: "destination",
        data,
        position: { x: 800, y },
      });
      topPosition += 100;
    });
    return nodes;
  }

  function getSourcesNodes() {
    let nodes: any = [];
    const canvasHeight = containerRef.current?.clientHeight;
    const listItemHeight = 120; // Adjust this value to the desired height of each list item
    const totalListItemsHeight = sources.length * listItemHeight;

    let topPosition = (canvasHeight - totalListItemsHeight) / 2;

    sources.forEach((data, index) => {
      const y = topPosition;
      nodes.push({
        id: `source-${index}`,
        type: "namespace",
        data,
        position: { x: 0, y },
      });
      topPosition += 100;
    });
    console.log({ nodes });
    return nodes;
  }

  return (
    <div ref={containerRef} style={{ width: "100%", height: "100%" }}>
      <ReactFlow
        nodes={namespaceNodes}
        edges={initialEdges}
        nodeTypes={nodeTypes}
        nodesDraggable={false}
        nodeOrigin={[0.4, 0.4]}
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
