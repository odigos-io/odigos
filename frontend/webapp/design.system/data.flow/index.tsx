"use client";
import React, { useEffect, useRef, useState } from "react";
import ReactFlow, {
  Background,
  Controls,
  useReactFlow,
  ReactFlowProvider,
} from "reactflow";
import CustomNode from "./keyval.middleware";
import NamespaceNode from "./namespace.node";
import DestinationNode from "./destination.node";
import "reactflow/dist/style.css";

const nodeTypes = {
  custom: CustomNode,
  namespace: NamespaceNode,
  destination: DestinationNode,
};

function KeyvalDataFlow({ sources, destinations, nodes }) {
  const [namespaceNodes, setNamespaceNodes] = useState([]);
  const [initialEdges, setInitialEdges] = useState([]);
  const containerRef = useRef(null);

  const { zoomTo, fitView } = useReactFlow();

  useEffect(() => {
    console.log({ nodes });
  }, [nodes]);

  useEffect(() => {
    setTimeout(() => {
      fitView();
      zoomTo(1);
    }, 100);
  }, [fitView, nodes]);

  // useEffect(() => {
  //   destinations && centerDestinationListVertically();
  // }, [destinations]);

  // function centerDestinationListVertically() {
  //   const canvasHeight = containerRef.current?.clientHeight;
  //   const listItemHeight = 120; // Adjust this value to the desired height of each list item
  //   const totalListItemsHeight = destinations.length * listItemHeight;

  //   let topPosition = (canvasHeight - totalListItemsHeight) / 2;

  //   const destinations_nodes = getDestinationNodes();
  //   const sources_nodes = getSourcesNodes();

  //   let nodes: any = [
  //     {
  //       id: "1",
  //       type: "custom",
  //       data: null,

  //       position: { x: 385, y: 300 },
  //     },
  //     ...destinations_nodes,
  //     ...sources_nodes,
  //   ];

  //   const destinations_edges = destinations_nodes.map((node, index) => {
  //     return {
  //       id: `edges-${node.id}`,
  //       source: "1",
  //       target: `destination-${index}`,
  //       animated: true,
  //       style: { stroke: "#96f3ff8e" },
  //       data: null,
  //     };
  //   });

  //   const sources_edges = sources_nodes.map((node, index) => {
  //     return {
  //       id: `edges-${node.id}`,
  //       source: `source-${index}`,
  //       target: "1",
  //       animated: true,
  //       style: { stroke: "#96f3ff8e" },
  //       data: null,
  //     };
  //   });

  //   setInitialEdges([...sources_edges, ...destinations_edges]);
  //   setNamespaceNodes(nodes);

  // }

  // function getDestinationNodes() {
  //   let nodes: any = [];
  //   const canvasHeight = containerRef.current?.clientHeight;
  //   const listItemHeight = 120; // Adjust this value to the desired height of each list item
  //   const totalListItemsHeight = destinations.length * listItemHeight;

  //   let topPosition = (canvasHeight - totalListItemsHeight) / 2;

  //   destinations.forEach((data, index) => {
  //     const y = topPosition;
  //     nodes.push({
  //       id: `destination-${index}`,
  //       type: "destination",
  //       data,
  //       position: { x: 800, y },
  //     });
  //     topPosition += 100;
  //   });
  //   return nodes;
  // }

  // function getSourcesNodes() {
  //   let nodes: any = [];
  //   const canvasHeight = containerRef.current?.clientHeight;
  //   const listItemHeight = 120; // Adjust this value to the desired height of each list item
  //   const totalListItemsHeight = sources.length * listItemHeight;

  //   let topPosition = (canvasHeight - totalListItemsHeight) / 2;

  //   sources.forEach((data, index) => {
  //     const y = topPosition;
  //     nodes.push({
  //       id: `source-${index}`,
  //       type: "namespace",
  //       data,
  //       position: { x: 0, y },
  //     });
  //     topPosition += 100;
  //   });
  //   return nodes;
  // }

  return (
    <div ref={containerRef} style={{ width: "100%", height: "100%" }}>
      <ReactFlow
        nodes={nodes}
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
