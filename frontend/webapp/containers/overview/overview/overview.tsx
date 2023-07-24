"use client";
import React, { useCallback, useMemo, useState } from "react";
import { KeyvalDataFlow, KeyvalLoader } from "@/design.system";
import { QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations, getSources } from "@/services/setup";
import { getEdges, groupSourcesNamespace, getNodes } from "./utils";
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

  const { data: sources } = useQuery([QUERIES.API_SOURCES], getSources);

  const sourcesNodes = useMemo(() => {
    const nodes = getNodes(
      containerHeight,
      groupSourcesNamespace(sources),
      "namespace",
      84,
      0,
      true
    );

    return nodes;
  }, [sources, containerHeight]);

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
      <KeyvalDataFlow
        nodes={[...destinationsNodes, ...sourcesNodes]}
        edges={edges}
      />
    </OverviewDataFlowWrapper>
  );
}
