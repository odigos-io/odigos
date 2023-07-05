import React from "react";
import { SetupSectionContainer } from "./setup.section.styled";
import { SetupHeader } from "../setup.header/setup.header";
import { SourceCard } from "@/components/index";
import { KeyvalDropDown } from "design.system";

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
