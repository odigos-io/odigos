"use client";
import React, { useState } from "react";
import { OVERVIEW } from "@/utils/constants";
import { OverviewHeader } from "@/components/overview";
import { SourcesContainerWrapper } from "./sources.styled";
import { NewSourceFlow } from "./new.source.flow";
import { ManageSources } from "./manage.sources";

export function SourcesContainer() {
  const [displayNewSourceFlow, setDisplayNewSourceFlow] = useState(false);

  function renderNewSourceFlow() {
    return <NewSourceFlow />;
  }

  function renderSources() {
    return <ManageSources setDisplayNewSourceFlow={setDisplayNewSourceFlow} />;
  }

  return (
    <SourcesContainerWrapper>
      <OverviewHeader
        title={OVERVIEW.MENU.SOURCES}
        onBackClick={
          displayNewSourceFlow ? () => setDisplayNewSourceFlow(false) : null
        }
      />
      {displayNewSourceFlow ? renderNewSourceFlow() : renderSources()}
    </SourcesContainerWrapper>
  );
}
