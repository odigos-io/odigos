"use client";
import Steps from "design.system/steps/steps";
import { LogoWrapper, SetupPageContainer } from "./setup.styled";
import Logo from "assets/logos/odigos-gradient.svg";
import { SetupHeader, SetupSection } from "containers/setup";

const STEPS = [
  {
    title: "Choose Source",
    status: "done",
  },
  {
    title: "Choose Destination",
    status: "active",
  },
  {
    title: "Create Connection",
    status: "disabled",
  },
];

export default function SetupPage() {
  return (
    <SetupPageContainer>
      <LogoWrapper>
        <Logo />
      </LogoWrapper>
      <br />
      <br />
      <br />
      <br />
      <br />
      <br />

      <Steps data={STEPS} />
      <SetupSection />
    </SetupPageContainer>
  );
}
