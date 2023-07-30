import React from "react";
import { useSectionData } from "@/hooks";
import { SourcesSectionWrapper, ButtonWrapper } from "./sources.styled";
import { SourcesSection } from "@/containers/setup/sources/sources.section";
import { KeyvalButton, KeyvalText } from "@/design.system";
import theme from "@/styles/palette";
import { SETUP } from "@/utils/constants";

export function NewSourceFlow() {
  const { sectionData, setSectionData, totalSelected } = useSectionData({});
  return (
    <SourcesSectionWrapper>
      <ButtonWrapper>
        <KeyvalText>{`${totalSelected} ${SETUP.SELECTED}`}</KeyvalText>
        <KeyvalButton style={{ width: 110 }}>
          <KeyvalText weight={600} color={theme.text.dark_button}>
            Connect
          </KeyvalText>
        </KeyvalButton>
      </ButtonWrapper>
      <SourcesSection
        sectionData={sectionData}
        setSectionData={setSectionData}
      />
    </SourcesSectionWrapper>
  );
}
