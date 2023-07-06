"use client";
import Steps from "@/design.system/steps/steps";
import { LogoWrapper, SetupPageContainer } from "./setup.styled";
import Logo from "@/assets/logos/odigos-gradient.svg";
import { SetupSection } from "@/containers/setup";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "react-query";
import { getNamespaces } from "@/services/setup";
import { QUERIES } from "@/utils/constants";

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
  const router = useRouter();
  const { isLoading, isError, isSuccess, data } = useQuery(
    [QUERIES.API_NAMESPACES],
    getNamespaces
  );

  useEffect(() => {
    console.log({ isLoading, data });
  }, [data]);

  if (isLoading) {
    return <div>Loading...</div>;
  }

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
