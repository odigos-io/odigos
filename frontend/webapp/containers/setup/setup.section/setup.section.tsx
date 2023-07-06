import React, { useEffect, useMemo, useState } from "react";
import {
  EmptyListWrapper,
  SetupContentWrapper,
  SetupSectionContainer,
} from "./setup.section.styled";
import { SetupHeader } from "../setup.header/setup.header";
import { SourcesList, SourcesOptionMenu } from "@/components/setup";
import { useQuery } from "react-query";
import { getApplication } from "@/services/setup";
import { QUERIES } from "@/utils/constants";
import Empty from "@/assets/images/empty-list.svg";
export function SetupSection({ namespaces }: any) {
  const [currentNamespace, setCurrentNamespace] = useState<any>(null);
  const [searchFilter, setSearchFilter] = useState<string>("");

  const namespacesList = useMemo(() => {
    return namespaces.map((item: any, index: number) => {
      return { id: index, label: item.name };
    });
  }, [namespaces]);

  const { data } = useQuery(
    [QUERIES.API_APPLICATIONS, currentNamespace],
    () => getApplication(currentNamespace.name),
    {
      // The query will not execute until the currentNamespace exists
      enabled: !!currentNamespace,
    }
  );

  useEffect(() => {
    !currentNamespace && setCurrentNamespace(namespaces[0]);
  }, [namespaces]);

  function getSourceData() {
    return searchFilter
      ? data?.applications.filter((item: any) =>
          item.name.toLowerCase().includes(searchFilter.toLowerCase())
        )
      : data?.applications;
  }

  return (
    <SetupSectionContainer>
      <SetupHeader />
      <SetupContentWrapper>
        <SourcesOptionMenu
          setCurrentItem={setCurrentNamespace}
          data={namespacesList}
          searchFilter={searchFilter}
          setSearchFilter={setSearchFilter}
        />
        {!data?.applications?.length ? (
          <EmptyListWrapper>
            <Empty />
          </EmptyListWrapper>
        ) : (
          <SourcesList data={getSourceData()} />
        )}
      </SetupContentWrapper>
    </SetupSectionContainer>
  );
}
