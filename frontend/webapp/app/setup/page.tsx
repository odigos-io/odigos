"use client";
import Steps from "@/design.system/steps/steps";
import {
  LogoWrapper,
  SetupPageContainer,
  StepListWrapper,
} from "./setup.styled";
import Logo from "@/assets/logos/odigos-gradient.svg";
import { SetupSection } from "@/containers/setup";
import { useState } from "react";
import { useQuery } from "react-query";
import { getNamespaces } from "@/services/setup";
import { QUERIES, SETUP } from "@/utils/constants";

const STEPS = [
  {
    title: SETUP.STEPS.CHOOSE_SOURCE,
    status: SETUP.STEPS.STATUS.ACTIVE,
  },
  {
    title: SETUP.STEPS.CHOOSE_DESTINATION,
    status: SETUP.STEPS.STATUS.DISABLED,
  },
  {
    title: SETUP.STEPS.CREATE_CONNECTION,
    status: SETUP.STEPS.STATUS.DISABLED,
  },
];

export default function SetupPage() {
  const [steps, setSteps] = useState(STEPS);

  const { isLoading, data } = useQuery([QUERIES.API_NAMESPACES], getNamespaces);

  if (isLoading) {
    return null;
  }

  return (
    <SetupPageContainer>
      <LogoWrapper>
        <Logo />
      </LogoWrapper>
      <StepListWrapper>
        <Steps data={steps} />
      </StepListWrapper>

      <SetupSection namespaces={data?.namespaces} />
    </SetupPageContainer>
  );
}
