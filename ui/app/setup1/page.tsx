"use client";
import Steps from "design.system/steps/steps";
import { SetupPageContainer } from "./setup.styled";

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

export default function Setup() {
  return (
    <SetupPageContainer>
      <div style={{ backgroundColor: "#000" }}>
        <Steps data={STEPS} />
      </div>
    </SetupPageContainer>
  );
}
