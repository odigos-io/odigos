"use client";
import Steps from "design.system/steps/steps";
import { SetupPageContainer } from "./setup.styled";
import Icon from "../../assets/icons/checked.svg";
import { KeyvalSVG } from "design.system";
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
        <KeyvalSVG path={Icon}>
          <Icon />
        </KeyvalSVG>
      </div>
    </SetupPageContainer>
  );
}
