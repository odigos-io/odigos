import React from "react";
import { SetupSectionContainer } from "./setup.section.styled";
import { SetupHeader } from "../setup.header/setup.header";
import { KeyvalDropDown } from "@/design.system";
import { SourceCard } from "@/components/setup/source.card/source.card";

export function SetupSection() {
  return (
    <SetupSectionContainer>
      <SetupHeader />
      {/* //option menu */}
      <SourceCard />
      <KeyvalDropDown />
    </SetupSectionContainer>
  );
}
