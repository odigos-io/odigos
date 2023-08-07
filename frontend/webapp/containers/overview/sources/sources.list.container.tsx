"use client";
import React from "react";
import { NOTIFICATION, OVERVIEW, QUERIES } from "@/utils/constants";
import { OverviewHeader } from "@/components/overview";
import { useNotification } from "@/hooks";
import { useQuery } from "react-query";
import { getSources } from "@/services";
import { NewSourcesList } from "@/containers/overview/sources/new.source.flow";
import { useRouter } from "next/navigation";

export function SourcesListContainer() {
  const { show, Notification } = useNotification();
  const router = useRouter();
  const { data: sources, refetch } = useQuery(
    [QUERIES.API_SOURCES],
    getSources
  );

  function onNewSourceSuccess() {
    setTimeout(() => {
      router.back();
      refetch();
    }, 1000);
    show({
      type: NOTIFICATION.SUCCESS,
      message: OVERVIEW.SOURCE_CREATED_SUCCESS,
    });
  }

  return (
    <>
      <OverviewHeader
        title={OVERVIEW.MENU.SOURCES}
        onBackClick={() => router.back()}
      />
      <NewSourcesList onSuccess={onNewSourceSuccess} sources={sources} />
      <Notification />
    </>
  );
}
