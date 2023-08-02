import React from "react";
import { useNotification, useSectionData } from "@/hooks";
import { SourcesSectionWrapper, ButtonWrapper } from "./sources.styled";
import { SourcesSection } from "@/containers/setup/sources/sources.section";
import { KeyvalButton, KeyvalText } from "@/design.system";
import theme from "@/styles/palette";
import { NOTIFICATION, OVERVIEW, SETUP } from "@/utils/constants";
import { useMutation } from "react-query";
import { setNamespaces } from "@/services";
import { SelectedSources } from "@/types/sources";

export function NewSourceFlow({ onSuccess, sources }) {
  const { sectionData, setSectionData, totalSelected } = useSectionData({});
  const { mutate } = useMutation((body: SelectedSources) =>
    setNamespaces(body)
  );
  const { show, Notification } = useNotification();

  function updateSectionDataWithSources() {
    const sourceNamesSet = new Set(sources.map((source) => source.name));
    const updatedSectionData: SelectedSources = {};

    for (const key in sectionData) {
      const { objects, ...rest } = sectionData[key];
      const updatedObjects = objects.map((item) => ({
        ...item,
        selected: item?.selected || sourceNamesSet.has(item.name),
      }));

      updatedSectionData[key] = {
        ...rest,
        objects: updatedObjects,
      };
    }

    mutate(updatedSectionData, {
      onSuccess,
      onError: ({ response }) => {
        const message = response?.data?.message || SETUP.ERROR;
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
        <KeyvalButton
          onClick={updateSectionDataWithSources}
          style={{ width: 110 }}
        >
          <KeyvalText weight={600} color={theme.text.dark_button}>
            {OVERVIEW.CONNECT}
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
