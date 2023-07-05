import React from "react";
import {
  SetupContentWrapper,
  SetupSectionContainer,
} from "./setup.section.styled";
import { SetupHeader } from "../setup.header/setup.header";
import { SourcesList, SourcesOptionMenu } from "@/components/setup";

export function SetupSection() {
  return (
    <SetupSectionContainer>
      <SetupHeader />
      <SetupContentWrapper>
        <SourcesOptionMenu />
        <SourcesList />
      </SetupContentWrapper>
    </SetupSectionContainer>
  );
}
