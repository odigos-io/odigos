"use client";
import React, { useState } from "react";
import { NOTIFICATION, OVERVIEW, QUERIES } from "@/utils/constants";
import { OverviewHeader } from "@/components/overview";
import { SourcesContainerWrapper } from "./sources.styled";
import { NewSourceFlow } from "./new.source.flow";
import { ManageSources } from "./manage.sources";
import { useNotification } from "@/hooks";
import { useQuery } from "react-query";
import { getSources } from "@/services";

export function SourcesContainer() {
  const [displayNewSourceFlow, setDisplayNewSourceFlow] = useState(false);
  const { show, Notification } = useNotification();

  const { data: sources, isLoading } = useQuery(
    [QUERIES.API_SOURCES],
    getSources
  );

  function onNewSourceSuccess() {
    setDisplayNewSourceFlow(false);
    show({
      type: NOTIFICATION.SUCCESS,
      message: OVERVIEW.SOURCE_CREATED_SUCCESS,
    });
  }

  function renderNewSourceFlow() {
    return <NewSourceFlow onSuccess={onNewSourceSuccess} sources={sources} />;
  }

  function renderSources() {
    return (
      <ManageSources
        setDisplayNewSourceFlow={setDisplayNewSourceFlow}
        sources={sources}
      />
    );
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
      <Notification />
    </SourcesContainerWrapper>
  );
}
