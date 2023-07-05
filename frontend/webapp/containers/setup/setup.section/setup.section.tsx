import React from "react";
import {
  SetupListWrapper,
  SetupSectionContainer,
} from "./setup.section.styled";
import { SetupHeader } from "../setup.header/setup.header";
import { SourcesList } from "@/components/setup";

export function SetupSection() {
  return (
    <SetupSectionContainer>
      <SetupHeader />
      <SetupListWrapper>
        <SourcesList />
      </SetupListWrapper>
    </SetupSectionContainer>
  );
}
