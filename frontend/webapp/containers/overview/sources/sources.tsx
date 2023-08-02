"use client";
import React from "react";
import { OVERVIEW, QUERIES, ROUTES } from "@/utils/constants";
import { OverviewHeader } from "@/components/overview";
import { SourcesContainerWrapper } from "./sources.styled";
import { ManageSources } from "./manage.sources";
import { useQuery } from "react-query";
import { getSources } from "@/services";
import { useRouter } from "next/navigation";

export function SourcesContainer() {
  const router = useRouter();
  const { data: sources } = useQuery([QUERIES.API_SOURCES], getSources);

  return (
    <SourcesContainerWrapper>
      <OverviewHeader title={OVERVIEW.MENU.SOURCES} />
      <ManageSources
        onAddClick={() => router.push(ROUTES.CREATE_SOURCE)}
        sources={sources}
      />
    </SourcesContainerWrapper>
  );
}
