import React, { useState } from "react";
import { styled } from "styled-components";
import { Back } from "@/assets/icons/overview";
import { ConnectionsIcons, CreateConnectionForm } from "@/components/setup";
import { DangerZone, KeyvalModal, KeyvalText } from "@/design.system";
import { SETUP } from "@/utils/constants";
import { ManageDestinationHeader } from "../manage.destination.header/manage.destination.header";
import { DestinationType } from "@/types/destinations";
import { ModalPositionX, ModalPositionY } from "@/design.system/modal/types";
import theme from "@/styles/palette";
import { useNotification } from "@/hooks";

interface ManageDestinationProps {
  destinationType: DestinationType;
  selectedDestination: any;
  onBackClick: () => void;
  onSubmit: (data: any) => void;
  onDelete?: () => void;
}

const BackButtonWrapper = styled.div`
  display: flex;
  align-items: center;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;

const FieldWrapper = styled.div`
  margin-top: 32px;
  width: 348px;
`;

function FormDangerZone({
  onDelete,
  data,
}: {
  onDelete: () => void;
  data: any;
}) {
  const [showModal, setShowModal] = useState(false);

  const modalConfig = {
    title: `Delete destination`,
    showHeader: true,
    showOverlay: true,
    positionX: ModalPositionX.center,
    positionY: ModalPositionY.center,
    padding: "20px",
    footer: {
      primaryBtnText: "I want to delete this destination",
      primaryBtnAction: () => {
        onDelete();
        setShowModal(false);
      },
    },
  };

  return (
    <>
      <FieldWrapper>
        <DangerZone
          title="Delete this destination"
          subTitle="This action cannot be undone. This will permanently delete the destination and all associated data."
          btnText="Delete"
          onClick={() => setShowModal(true)}
        />
      </FieldWrapper>
      <KeyvalModal
        show={showModal}
        setShow={() => setShowModal(false)}
        config={modalConfig}
      >
        <br />
        <ConnectionsIcons
          icon={data?.image_url}
          imageStyle={{ border: "solid 1px #ededed" }}
        />
        <br />
        <KeyvalText color={theme.text.primary} size={20} weight={600}>
          {`Delete ${data?.name}`}
        </KeyvalText>
      </KeyvalModal>
    </>
  );
}

export function ManageDestination({
  destinationType,
  selectedDestination,
  onBackClick,
  onSubmit,
  onDelete,
}: ManageDestinationProps) {
  return (
    <>
      <BackButtonWrapper onClick={onBackClick}>
        <Back width={14} />
        <KeyvalText size={14}>{SETUP.BACK}</KeyvalText>
      </BackButtonWrapper>
      <ManageDestinationHeader data={selectedDestination} />
      <CreateConnectionForm
        fields={destinationType?.fields}
        destinationNameValue={selectedDestination?.name}
        dynamicFieldsValues={selectedDestination?.fields}
        checkboxValues={selectedDestination?.signals}
        supportedSignals={selectedDestination?.supported_signals}
        onSubmit={(data) => onSubmit(data)}
      />
      {onDelete && (
        <FormDangerZone onDelete={onDelete} data={selectedDestination} />
      )}
    </>
  );
}
