import React from "react";
import { useNotification, useSectionData } from "@/hooks";
import { SourcesSectionWrapper, ButtonWrapper } from "./sources.styled";
import { SourcesSection } from "@/containers/setup/sources/sources.section";
import { KeyvalButton, KeyvalText } from "@/design.system";
import theme from "@/styles/palette";
import { NOTIFICATION, SETUP } from "@/utils/constants";
import { useMutation } from "react-query";
import { setNamespaces } from "@/services";

export function NewSourceFlow() {
  const { sectionData, setSectionData, totalSelected } = useSectionData({});
  const { mutate } = useMutation((body) => setNamespaces(body));
  const { show, Notification } = useNotification();

  function handleNewSource() {
    mutate(sectionData, {
      onSuccess: () => {
        setSectionData({});
      },
      onError: ({ response }) => {
        const message = response?.data?.message || SETUP.ERROR;
        console.log({ response });
        show({
          type: NOTIFICATION.ERROR,
          message,
        });
      },
    });
  }

  return (
    <SourcesSectionWrapper>
      <ButtonWrapper>
        <KeyvalText>{`${totalSelected} ${SETUP.SELECTED}`}</KeyvalText>
        <KeyvalButton onClick={handleNewSource} style={{ width: 110 }}>
          <KeyvalText weight={600} color={theme.text.dark_button}>
            Connect
          </KeyvalText>
        </KeyvalButton>
      </ButtonWrapper>
      <SourcesSection
        sectionData={sectionData}
        setSectionData={setSectionData}
      />
      <Notification />
    </SourcesSectionWrapper>
  );
}
