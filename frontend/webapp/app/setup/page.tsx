"use client";
import Steps from "@/design.system/steps/steps";
import { LogoWrapper, SetupPageContainer } from "./setup.styled";
import Logo from "@/assets/logos/odigos-gradient.svg";
import { SetupSection } from "@/containers/setup";
import { useEffect } from "react";

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
  // useEffect(() => {
  //   test();
  // }, []);

  // async function test() {
  //   const response = await fetch("/api/config");
  //   console.log({ response });

  //   const data = await response.json();
  //   console.log({ data });
  // }

  return (
    <SetupPageContainer>
      <LogoWrapper>
        <Logo />
      </LogoWrapper>
      <br />
      <br />
      <br />
      <br />
      <Steps data={STEPS} />
      <br />
      <br />
      <SetupSection />
    </SetupPageContainer>
  );
}
