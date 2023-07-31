"use client";
import React, { useState } from "react";
import { NOTIFICATION, OVERVIEW } from "@/utils/constants";
import { OverviewHeader } from "@/components/overview";
import { SourcesContainerWrapper } from "./sources.styled";
import { NewSourceFlow } from "./new.source.flow";
import { ManageSources } from "./manage.sources";
import { useNotification } from "@/hooks";

export function SourcesContainer() {
  const [displayNewSourceFlow, setDisplayNewSourceFlow] = useState(false);
  const { show, Notification } = useNotification();
  function onNewSourceSuccess() {
    setDisplayNewSourceFlow(false);
    show({
      type: NOTIFICATION.SUCCESS,
      message: OVERVIEW.SOURCE_CREATED_SUCCESS,
    });
  }

  function renderNewSourceFlow() {
    return <NewSourceFlow onSuccess={onNewSourceSuccess} />;
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
      <Notification />
    </SourcesContainerWrapper>
  );
}
