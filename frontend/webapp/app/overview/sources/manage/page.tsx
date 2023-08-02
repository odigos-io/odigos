"use client";
import { ManageSourceHeader } from "@/components/overview/sources/manage.source.header/manage.source.header";
import { getSources } from "@/services";
import { QUERIES, SETUP } from "@/utils/constants";
import { useRouter, useSearchParams } from "next/navigation";
import React, { useEffect, useState } from "react";
import { useQuery } from "react-query";
import { ManageSourcePageContainer, BackButtonWrapper } from "./styled";
import { LANGUAGES_LOGOS } from "@/assets/images";
import { Back } from "@/assets/icons/overview";
import { KeyvalText } from "@/design.system";
import { ManagedSource } from "@/types/sources";
import { DeleteSource } from "@/components/overview";

const SOURCE = "source";

export default function ManageSourcePage() {
  const [currentSource, setCurrentSource] = useState<ManagedSource | null>(
    null
  );
  const searchParams = useSearchParams();
  const router = useRouter();
  const { data: sources } = useQuery([QUERIES.API_SOURCES], getSources);

  useEffect(onPageLoad, [sources]);

  function onPageLoad() {
    const search = searchParams.get(SOURCE);
    const source = sources?.find((item) => item.name === search);
    source && setCurrentSource(source);
  }

  return (
    <ManageSourcePageContainer>
      <BackButtonWrapper onClick={() => router.back()}>
        <Back width={14} />
        <KeyvalText size={14}>{SETUP.BACK}</KeyvalText>
      </BackButtonWrapper>
      {currentSource && (
        <ManageSourceHeader
          display_name={currentSource?.name}
          image_url={
            LANGUAGES_LOGOS[currentSource?.languages?.[0].language || ""]
          }
        />
      )}
      <DeleteSource
        onDelete={() => {}}
        name={currentSource?.name}
        image_url={
          LANGUAGES_LOGOS[currentSource?.languages?.[0].language || ""]
        }
      />
    </ManageSourcePageContainer>
  );
}
