import React, { useMemo } from "react";
import { useMutation, useQuery } from "react-query";
import { CreateConnectionForm, QuickHelp } from "@/components/setup";
import {
  CreateConnectionContainer,
  LoaderWrapper,
} from "./connection.section.styled";
import { getDestination, setDestination } from "@/services/setup";
import { QUERIES } from "@/utils/constants";
import { KeyvalLoader } from "@/design.system";

export interface DestinationBody {
  name: string;
  type?: string;
  signals: {
    [key: string]: boolean;
  };
  fields: {
    [key: string]: string;
  };
}

export function ConnectionSection({ sectionData }) {
  const { isLoading, data } = useQuery([QUERIES.API_DESTINATION_TYPE], () =>
    getDestination(sectionData.type)
  );

  const { mutate } = useMutation((body) => setDestination(body));

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

  function createDestination(formData: DestinationBody) {
    const { type } = sectionData;
    const body: any = {
      type,
      ...formData,
    };

    mutate(body, {
      onSuccess: (data) => console.log("onSuccess", { data }), //TODO: redirect to next step
      onError: (error) => {
        console.log("onError", { error });
      },
    });
  }

  if (isLoading)
    return (
      <LoaderWrapper>
        <KeyvalLoader />
      </LoaderWrapper>
    );

  return (
    <CreateConnectionContainer>
      {data?.fields && (
        <CreateConnectionForm
          fields={data?.fields}
          onSubmit={createDestination}
          supportedSignals={sectionData?.supported_signals}
        />
      )}
      {videoList?.length > 0 && <QuickHelp data={videoList} />}
    </CreateConnectionContainer>
  );
}
