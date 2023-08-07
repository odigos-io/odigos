"use client";
import { ManageSourceHeader } from "@/components/overview/sources/manage.source.header/manage.source.header";
import { ACTION, NOTIFICATION, OVERVIEW, SETUP } from "@/utils/constants";
import { useRouter, useSearchParams } from "next/navigation";
import React, { useEffect, useState } from "react";
import { useMutation } from "react-query";
import {
  ManageSourcePageContainer,
  BackButtonWrapper,
  FieldWrapper,
  SaveSourceButtonWrapper,
} from "./sources.styled";
import { LANGUAGES_LOGOS } from "@/assets/images";
import { Back } from "@/assets/icons/overview";
import {
  KeyvalButton,
  KeyvalInput,
  KeyvalLoader,
  KeyvalText,
} from "@/design.system";
import { DeleteSource } from "@/components/overview";
import { deleteSource, getSource, patchSources } from "@/services/sources";
import { useNotification } from "@/hooks";
import theme from "@/styles/palette";
import { ManagedSource } from "@/types/sources";

const NAME = "name";
const KIND = "kind";
const NAMESPACE = "namespace";

export function UpdateSourceForm() {
  const [inputValue, setInputValue] = useState("");
  const [currentSource, setCurrentSource] = useState<ManagedSource>();

  const searchParams = useSearchParams();
  const router = useRouter();
  const { show, Notification } = useNotification();

  const { mutate: handleDeleteSource } = useMutation(() =>
    deleteSource(
      currentSource?.namespace || "",
      currentSource?.kind || "",
      currentSource?.name || ""
    )
  );

  const { mutate: editSource } = useMutation(() =>
    patchSources(
      currentSource?.namespace || "",
      currentSource?.kind || "",
      currentSource?.name || "",
      { reported_name: inputValue }
    )
  );
  useEffect(() => {
    onPageLoad();
  }, [searchParams]);

  useEffect(() => {
    setInputValue(currentSource?.reported_name || "");
  }, [currentSource]);

  async function onPageLoad() {
    const name = searchParams.get(NAME) || "";
    const kind = searchParams.get(KIND) || "";
    const namespace = searchParams.get(NAMESPACE) || "";

    const currentSource = await getSource(namespace, kind, name);
    setCurrentSource(currentSource);
  }

  function onSaveClick(newName) {
    editSource(newName, {
      onError,
      onSuccess: () =>
        show({
          type: NOTIFICATION.SUCCESS,
          message: OVERVIEW.SOURCE_UPDATE_SUCCESS,
        }),
    });
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
    }, 1000);
    show({
      type: NOTIFICATION.SUCCESS,
      message: OVERVIEW.SOURCE_DELETED_SUCCESS,
    });
  }

  function onDelete() {
    handleDeleteSource(undefined, {
      onSuccess,
      onError,
    });
  }

  if (!currentSource) {
    return <KeyvalLoader />;
  }

  return (
    <ManageSourcePageContainer>
      <BackButtonWrapper onClick={() => router.back()}>
        <Back width={14} />
        <KeyvalText size={14}>{SETUP.BACK}</KeyvalText>
      </BackButtonWrapper>
      {currentSource && (
        <ManageSourceHeader
          image_url={
            LANGUAGES_LOGOS[currentSource?.languages?.[0].language || ""]
          }
        />
      )}
      <FieldWrapper>
        <KeyvalInput
          label={OVERVIEW.REPORTED_NAME}
          value={inputValue}
          onChange={(e) => setInputValue(e)}
        />
      </FieldWrapper>
      <SaveSourceButtonWrapper>
        <KeyvalButton disabled={!inputValue} onClick={onSaveClick}>
          <KeyvalText color={theme.colors.dark_blue} size={14} weight={600}>
            {ACTION.SAVE}
          </KeyvalText>
        </KeyvalButton>
      </SaveSourceButtonWrapper>
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
