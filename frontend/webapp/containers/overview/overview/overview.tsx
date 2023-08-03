"use client";
import React, { useCallback, useMemo, useState } from "react";
import { KeyvalDataFlow, KeyvalLoader } from "@/design.system";
import { QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations, getSources } from "@/services";
import { getEdges, groupSourcesNamespace, getNodes } from "./utils";
import { OverviewDataFlowWrapper } from "./overview.styled";

const NAMESPACE_NODE_HEIGHT = 84;
const NAMESPACE_NODE_POSITION = 0;
const DESTINATION_NODE_HEIGHT = 136;
const DESTINATION_NODE_POSITION = 800;

export function OverviewContainer() {
  const [containerHeight, setContainerHeight] = useState(0);

  const containerRef = useCallback((node: HTMLDivElement) => {
    if (node !== null) {
      setContainerHeight(node.getBoundingClientRect().height);
    }
  }, []);

  const { data: destinations } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  const { data: sources } = useQuery([QUERIES.API_SOURCES], getSources);

  const sourcesNodes = useMemo(() => {
    const nodes = getNodes(
      containerHeight,
      groupSourcesNamespace(sources),
      "namespace",
      NAMESPACE_NODE_HEIGHT,
      NAMESPACE_NODE_POSITION,
      true
    );

    return nodes;
  }, [sources, containerHeight]);

  const destinationsNodes = useMemo(
    () =>
      getNodes(
        containerHeight,
        destinations,
        "destination",
        destinations?.length > 1
          ? DESTINATION_NODE_HEIGHT
          : NAMESPACE_NODE_HEIGHT,
        DESTINATION_NODE_POSITION
      ),
    [destinations, containerHeight]
  );

  const edges = useMemo(() => {
    if (!destinationsNodes || !sourcesNodes) return [];

    return getEdges(destinationsNodes, sourcesNodes);
  }, [destinationsNodes, sourcesNodes]);

  if (!destinationsNodes || !sourcesNodes) {
    return <KeyvalLoader />;
  }

  return (
    <OverviewDataFlowWrapper ref={containerRef}>
      <KeyvalDataFlow
        nodes={[...destinationsNodes, ...sourcesNodes]}
        edges={edges}
      />
    </OverviewDataFlowWrapper>
  );
}
