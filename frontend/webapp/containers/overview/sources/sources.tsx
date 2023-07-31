"use client";
import React, { useState } from "react";
import { KeyvalLoader } from "@/design.system";
import { OVERVIEW, QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getSources } from "@/services";
import {
  OverviewHeader,
  SourcesActionMenu,
  SourcesManagedList,
} from "@/components/overview";
import { SourcesContainerWrapper, MenuWrapper } from "./sources.styled";
import { NewSourceFlow } from "./new.source.flow";

export function SourcesContainer() {
  const [displayNewSourceFlow, setDisplayNewSourceFlow] = useState(false);

  const {
    data: sources,
    refetch,
    isLoading,
  } = useQuery([QUERIES.API_SOURCES], getSources);

  function renderNewSourceFlow() {
    return <NewSourceFlow />;
  }

  function renderSources() {
    return (
      <>
        <MenuWrapper>
          <SourcesActionMenu onAddClick={() => setDisplayNewSourceFlow(true)} />
        </MenuWrapper>
        <SourcesManagedList data={sources} />
      </>
    );
  }

  if (isLoading) {
    return <KeyvalLoader />;
  }

  return (
    <SourcesContainerWrapper>
      <OverviewHeader title={OVERVIEW.MENU.SOURCES} />
      {displayNewSourceFlow ? renderNewSourceFlow() : renderSources()}
    </SourcesContainerWrapper>
  );
}
