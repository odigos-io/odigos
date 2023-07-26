import React from "react";
import { styled } from "styled-components";
import { Back } from "@/assets/icons/overview";
import { CreateConnectionForm } from "@/components/setup";
import { KeyvalText } from "@/design.system";
import { SETUP } from "@/utils/constants";
import { ManageDestinationHeader } from "../manage.destination.header/manage.destination.header";

interface DestinationType {
  fields: any;
  display_name: string;
  image_url: string;
  id: string;
}

interface ManageDestinationProps {
  destinationType: DestinationType;
  selectedDestination: any;
  onBackClick: () => void;
  onSubmit: (data: any) => void; // Replace 'any' with the actual type for the form data
}

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
}: ManageDestinationProps) {
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
        checkboxValues={selectedDestination?.signals}
        supportedSignals={
          selectedDestination?.destination_type?.supported_signals
        }
        onSubmit={(data) => onSubmit(data)}
      />
    </ManageDestinationWrapper>
  );
}
