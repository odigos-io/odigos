"use client";
import React, { useEffect, useState } from "react";
import { KeyvalLoader } from "@/design.system";
import { NOTIFICATION, OVERVIEW, QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getSources } from "@/services";
import { OverviewHeader } from "@/components/overview";
import { useNotification } from "@/hooks";
import { SourcesContainerWrapper } from "./sources.styled";

export function SourcesContainer() {
  const { show, Notification } = useNotification();

  const {
    data: sources,
    refetch,
    isLoading,
  } = useQuery([QUERIES.API_SOURCES], getSources);

  useEffect(() => {
    console.log({ sources });
  }, [sources]);

  function onSuccess(message = OVERVIEW.DESTINATION_UPDATE_SUCCESS) {
    refetch();

    show({
      type: NOTIFICATION.SUCCESS,
      message,
    });
  }

  function onError({ response }) {
    const message = response?.data?.message;
    show({
      type: NOTIFICATION.ERROR,
      message,
    });
  }

  if (isLoading) {
    return <KeyvalLoader />;
  }

  return (
    <SourcesContainerWrapper>
      <OverviewHeader title={OVERVIEW.MENU.SOURCES} />
      <Notification />
    </SourcesContainerWrapper>
  );
}
