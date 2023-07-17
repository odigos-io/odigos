import { getDestination } from "@/services/setup";
import { QUERIES } from "@/utils/constants";
import React, { useEffect } from "react";
import { useQuery } from "react-query";

export function ConnectionSection({ sectionData }) {
  console.log({ sectionData });
  const { isLoading, data } = useQuery([QUERIES.API_DESTINATION_TYPE], () =>
    getDestination(sectionData.type)
  );

  useEffect(() => {
    console.log({ data });
  }, [data]);

  return <>Connection</>;
}
