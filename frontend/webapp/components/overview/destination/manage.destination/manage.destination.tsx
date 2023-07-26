import { Back } from "@/assets/icons/overview";
import { CreateConnectionForm } from "@/components/setup";
import { KeyvalText } from "@/design.system";
import { SETUP } from "@/utils/constants";
import React, { useMemo } from "react";
import { styled } from "styled-components";
import { ManageDestinationHeader } from "../manage.destination.header/manage.destination.header";

const ManageDestinationWrapper = styled.div`
  padding: 32px;
`;

const BackButtonWrapper = styled.div`
  display: flex;
  align-items: center;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;

export function ManageDestination({
  destinationType: { fields },
  selectedDestination,
  onBackClick,
  onSubmit,
}) {
  return (
    <ManageDestinationWrapper>
      <BackButtonWrapper onClick={onBackClick}>
        <Back width={14} />
        <KeyvalText size={14}>{SETUP.BACK}</KeyvalText>
      </BackButtonWrapper>
      <ManageDestinationHeader data={selectedDestination} />
      <CreateConnectionForm
        fields={fields}
        destinationNameValue={selectedDestination?.name}
        dynamicFieldsValues={selectedDestination?.fields}
        signalsValues={selectedDestination?.signals}
        supportedSignals={
          selectedDestination?.destination_type?.supported_signals
        }
        onSubmit={(data) => onSubmit(data)}
      />
    </ManageDestinationWrapper>
  );
}
