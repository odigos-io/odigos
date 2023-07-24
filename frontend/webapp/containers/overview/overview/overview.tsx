"use client";
import React, { useCallback, useMemo, useState } from "react";
import { KeyvalFlow, KeyvalLoader } from "@/design.system";
import { QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services/setup";
import { getEdges, getNodes } from "./utils";
import { OverviewDataFlowWrapper } from "./overview.styled";

export function OverviewContainer() {
  const [containerHeight, setContainerHeight] = useState(0);

  const containerRef = useCallback((node: HTMLDivElement) => {
    if (node !== null) {
      setContainerHeight(node.getBoundingClientRect().height);
    }
  }, []);

  const { isLoading, data: destinations } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  const { data: sources } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  const sourcesNodes = useMemo(
    () => getNodes(containerHeight, sources, "namespace", 84, 0, true),
    [sources, containerHeight]
  );

  const destinationsNodes = useMemo(
    () => getNodes(containerHeight, destinations, "destination", 136, 800),
    [destinations, containerHeight]
  );

  const edges = useMemo(() => {
    if (!destinationsNodes || !sourcesNodes) return [];

    return getEdges(destinationsNodes, sourcesNodes);
  }, [destinationsNodes, sourcesNodes]);

  if (isLoading || !destinationsNodes || !sourcesNodes) {
    return <KeyvalLoader />;
  }

  return (
    <OverviewDataFlowWrapper ref={containerRef}>
      <KeyvalFlow
        nodes={[...destinationsNodes, ...sourcesNodes]}
        edges={edges}
      />
    </OverviewDataFlowWrapper>
  );
}
