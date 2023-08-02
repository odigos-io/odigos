"use client";
import React, { useEffect, useMemo, useState } from "react";
import { QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getNamespaces } from "@/services";
import { SourcesActionMenu, SourcesManagedList } from "@/components/overview";
import { MenuWrapper } from "./sources.styled";
import { ManagedSource, Namespace } from "@/types/sources";

const DEFAULT_FILTER = { name: "default", selected: false, totalApps: 0 };

export function ManageSources({ setDisplayNewSourceFlow, sources }) {
  const [searchFilter, setSearchFilter] = useState<string>("");
  const [currentNamespace, setCurrentNamespace] =
    useState<Namespace>(DEFAULT_FILTER);

  const { data: namespaces } = useQuery(
    [QUERIES.API_NAMESPACES],
    getNamespaces
  );

  useEffect(() => {
    setSearchFilter("");
  }, [currentNamespace]);

  const namespacesList = useMemo(
    () =>
      namespaces?.namespaces?.map((item: Namespace, index: number) => ({
        id: index,
        label: item.name,
      })),
    [namespaces]
  );

  function filterByNamespace() {
    return currentNamespace
      ? sources?.filter(
          (item: ManagedSource) => item.namespace === currentNamespace.name
        )
      : sources;
  }

  function filterBySearchQuery(data: ManagedSource[]) {
    return searchFilter
      ? data?.filter((item: ManagedSource) =>
          item.name.toLowerCase().includes(searchFilter.toLowerCase())
        )
      : data;
  }

  function filterSources() {
    let data = filterByNamespace();
    return filterBySearchQuery(data);
  }

  return (
    <>
      <MenuWrapper>
        <SourcesActionMenu
          searchFilter={searchFilter}
          setSearchFilter={setSearchFilter}
          data={namespacesList}
          onAddClick={() => setDisplayNewSourceFlow(true)}
          setCurrentItem={setCurrentNamespace}
        />
      </MenuWrapper>
      <SourcesManagedList data={filterSources()} />
    </>
  );
}
