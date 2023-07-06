import React, { useEffect, useState } from "react";
import {
  SetupContentWrapper,
  SetupSectionContainer,
} from "./setup.section.styled";
import { SetupHeader } from "../setup.header/setup.header";
import { SourcesList, SourcesOptionMenu } from "@/components/setup";
import { useQuery } from "react-query";
import { getApplication } from "@/services/setup";
import { QUERIES } from "@/utils/constants";

export function SetupSection({ namespaces }: any) {
  const [currentNamespace, setCurrentNamespace] = useState<any>(null);

  const { data } = useQuery(
    [QUERIES.API_APPLICATIONS],
    () => getApplication(currentNamespace.name),
    {
      // The query will not execute until the currentNamespace exists
      enabled: !!currentNamespace,
    }
  );

  useEffect(() => {
    if (!currentNamespace) {
      setCurrentNamespace(namespaces[0]);
    }
  }, [namespaces]);

  // useEffect(() => {
  //   console.log({ data });
  // }, [data]);

  return (
    <SetupSectionContainer>
      <SetupHeader />
      <SetupContentWrapper>
        <SourcesOptionMenu />
        <SourcesList data={data?.applications} />
      </SetupContentWrapper>
    </SetupSectionContainer>
  );
}
