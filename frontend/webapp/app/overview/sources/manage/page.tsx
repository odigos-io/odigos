"use client";
import { ManageSourceHeader } from "@/components/overview/sources/manage.source.header/manage.source.header";
import { getSources } from "@/services";
import {
  NOTIFICATION,
  OVERVIEW,
  QUERIES,
  ROUTES,
  SETUP,
} from "@/utils/constants";
import { useRouter, useSearchParams } from "next/navigation";
import React, { useEffect, useState } from "react";
import { useMutation, useQuery } from "react-query";
import { ManageSourcePageContainer, BackButtonWrapper } from "./styled";
import { LANGUAGES_LOGOS } from "@/assets/images";
import { Back } from "@/assets/icons/overview";
import { KeyvalText } from "@/design.system";
import { ManagedSource } from "@/types/sources";
import { DeleteSource } from "@/components/overview";
import { deleteSource } from "@/services/sources";
import { useNotification } from "@/hooks";

const SOURCE = "source";

export default function ManageSourcePage() {
  const [currentSource, setCurrentSource] = useState<ManagedSource | null>(
    null
  );
  const searchParams = useSearchParams();
  const router = useRouter();
  const { data: sources, refetch } = useQuery(
    [QUERIES.API_SOURCES],
    getSources
  );
  const { show, Notification } = useNotification();
  const { mutate } = useMutation(() =>
    deleteSource(
      currentSource?.namespace || "",
      currentSource?.kind || "",
      currentSource?.name || ""
    )
  );
  useEffect(onPageLoad, [sources]);

  useEffect(() => {
    console.log({ currentSource });
  }, [currentSource]);

  function onPageLoad() {
    const search = searchParams.get(SOURCE);
    const source = sources?.find((item) => item.name === search);
    source && setCurrentSource(source);
  }
  function onError({ response }) {
    const message = response?.data?.message;
    show({
      type: NOTIFICATION.ERROR,
      message,
    });
  }

  function onSuccess() {
    setTimeout(() => {
      router.back();
      refetch();
    }, 1000);
    show({
      type: NOTIFICATION.SUCCESS,
      message: OVERVIEW.SOURCE_DELETED_SUCCESS,
    });
  }
  function onDelete() {
    mutate(undefined, {
      onSuccess,
      onError,
    });
  }

  return (
    <ManageSourcePageContainer>
      <BackButtonWrapper onClick={() => router.back()}>
        <Back width={14} />
        <KeyvalText size={14}>{SETUP.BACK}</KeyvalText>
      </BackButtonWrapper>
      {currentSource && (
        <ManageSourceHeader
          name={currentSource?.name}
          image_url={
            LANGUAGES_LOGOS[currentSource?.languages?.[0].language || ""]
          }
        />
      )}
      <DeleteSource
        onDelete={onDelete}
        name={currentSource?.name}
        image_url={
          LANGUAGES_LOGOS[currentSource?.languages?.[0].language || ""]
        }
      />
      <Notification />
    </ManageSourcePageContainer>
  );
}
