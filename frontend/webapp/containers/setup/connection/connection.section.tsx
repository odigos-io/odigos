import React, { useMemo } from "react";
import { useQuery } from "react-query";
import { CreateConnectionForm, QuickHelp } from "@/components/setup";
import { CreateConnectionContainer } from "./connection.section.styled";
import { getDestination } from "@/services/setup";
import { QUERIES } from "@/utils/constants";

export function ConnectionSection({ sectionData }) {
  const { isLoading, data } = useQuery([QUERIES.API_DESTINATION_TYPE], () =>
    getDestination(sectionData.type)
  );

  const videoList = useMemo(
    () =>
      data?.fields
        ?.filter((field) => field?.video_url)
        ?.map((field) => ({
          name: field.display_name,
          src: field.video_url,
        })),
    [data]
  );

  if (isLoading) return <div>Loading...</div>;

  return (
    <CreateConnectionContainer>
      {data?.fields && <CreateConnectionForm fields={data?.fields} />}
      {videoList?.length > 0 && <QuickHelp data={videoList} />}
    </CreateConnectionContainer>
  );
}
