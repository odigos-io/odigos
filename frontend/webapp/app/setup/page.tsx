"use client";
import Steps from "@/design.system/steps/steps";
import {
  LogoWrapper,
  SetupPageContainer,
  StepListWrapper,
} from "./setup.styled";
import Logo from "@/assets/logos/odigos-gradient.svg";
import { SetupSection } from "@/containers/setup";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "react-query";
import { getNamespaces } from "@/services/setup";
import { QUERIES } from "@/utils/constants";

const STEPS = [
  {
    title: "Choose Source",
    status: "active",
  },
  {
    title: "Choose Destination",
    status: "disabled",
  },
  {
    title: "Create Connection",
    status: "disabled",
  },
];

export default function SetupPage() {
  const [steps, setSteps] = useState(STEPS);

  const { isLoading, isError, isSuccess, data } = useQuery(
    [QUERIES.API_NAMESPACES],
    getNamespaces
  );

  if (isLoading) {
    return <div>Loading...</div>;
  }
  console.log({ data });
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
