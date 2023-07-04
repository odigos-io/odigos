import React from "react";
import { SetupSectionContainer } from "./setup.section.styled";
import { SetupHeader } from "../setup.header/setup.header";
import { SourceCard } from "@/components/index";

export function SetupSection() {
  return (
    <SetupSectionContainer>
      <SetupHeader />
      {/* //option menu */}
      <SourceCard />
    </SetupSectionContainer>
  );
}
