import { Back } from "@/assets/icons/overview";
import { CreateConnectionForm } from "@/components/setup";
import { KeyvalText } from "@/design.system";
import { SETUP } from "@/utils/constants";
import React from "react";
import { styled } from "styled-components";
import { ManageDestinationHeader } from "../manage.destination.header/manage.destination.header";

const ManageDestinationWrapper = styled.div`
  padding: 32px;
`;

export const BackButtonWrapper = styled.div`
  display: flex;
  align-items: center;

  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;

export function ManageDestination({
  data: { fields },
  supportedSignals,
  onBackClick,
}) {
  console.log({ fields, supportedSignals });
  return (
    <ManageDestinationWrapper>
      <BackButtonWrapper onClick={onBackClick}>
        <Back width={14} />
        <KeyvalText size={14}>{SETUP.BACK}</KeyvalText>
      </BackButtonWrapper>
      <ManageDestinationHeader />
      <CreateConnectionForm
        fields={fields}
        supportedSignals={supportedSignals}
        onSubmit={(data) => console.log({ data })}
      />
    </ManageDestinationWrapper>
  );
}
