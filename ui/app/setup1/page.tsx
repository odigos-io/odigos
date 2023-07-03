"use client";
import Steps from "design.system/steps/steps";
import { SetupPageContainer } from "./setup.styled";

export default function Setup() {
  return (
    <SetupPageContainer>
      Setup
      <div>
        <Steps />
      </div>
    </SetupPageContainer>
  );
}
