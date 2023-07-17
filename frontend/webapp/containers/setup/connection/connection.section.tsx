import { CreateConnectionForm } from "@/components/setup";
import { getDestination } from "@/services/setup";
import { QUERIES } from "@/utils/constants";
import React, { useEffect } from "react";
import { useQuery } from "react-query";
import { CreateConnectionContainer } from "./connection.section.styled";
import { QuickHelp } from "@/components/setup/connection/quick.help/quick.help";

export function ConnectionSection({ sectionData }) {
  const { isLoading, data } = useQuery([QUERIES.API_DESTINATION_TYPE], () =>
    getDestination(sectionData.type)
  );

  if (isLoading) return <div>Loading...</div>;

  return (
    <CreateConnectionContainer>
      <CreateConnectionForm fields={data?.fields} />
      <QuickHelp />
    </CreateConnectionContainer>
  );
}
