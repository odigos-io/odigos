"use client";
import React, { useCallback, useMemo, useState } from "react";
import { KeyvalFlow, KeyvalLoader } from "@/design.system";
import { OVERVIEW, QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services/setup";
import { getDestinationNodes, getSourcesNodes } from "./utils";

export function OverviewContainer() {
  const [containerHeight, setContainerHeight] = useState(0);

  const containerRef = useCallback((node) => {
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

  const sourcesNodes = useMemo(() => {
    const data = getSourcesNodes(containerHeight, sources);
    return data;
  }, [sources, containerHeight]);

  const destinationsNodes = useMemo(() => {
    const data = getDestinationNodes(containerHeight, destinations);
    return data;
  }, [destinations, containerHeight]);

  if (isLoading || !destinationsNodes || !sourcesNodes) {
    return <KeyvalLoader />;
  }

  return (
    <div style={{ width: "100%", height: "100%" }} ref={containerRef}>
      <KeyvalFlow
        nodes={[...destinationsNodes, ...sourcesNodes]}
        destinations={destinations}
        sources={sources}
      />
    </div>
  );
}
