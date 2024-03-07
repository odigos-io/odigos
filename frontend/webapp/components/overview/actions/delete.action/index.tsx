'use client';
import React, { useState } from 'react';
import { OVERVIEW } from '@/utils';
import { ACTION_ICONS } from '@/assets';
import { styled } from 'styled-components';
import { DangerZone, KeyvalModal, KeyvalText } from '@/design.system';
import { ModalPositionX, ModalPositionY } from '@/design.system/modal/types';

const FieldWrapper = styled.div`
  div {
    width: 354px;
  }
`;

export function DeleteAction({
  onDelete,
  name,
  type,
}: {
  onDelete: () => void;
  name: string | undefined;
  type?: string;
}) {
  const [showModal, setShowModal] = useState(false);

  const modalConfig = {
    title: OVERVIEW.DELETE_ACTION,
    showHeader: true,
    showOverlay: true,
    positionX: ModalPositionX.center,
    positionY: ModalPositionY.center,
    padding: '20px',
    footer: {
      primaryBtnText: OVERVIEW.CONFIRM_DELETE_ACTION,
      primaryBtnAction: () => {
        setShowModal(false);
        onDelete();
      },
    },
  };
  const ActionIcon = ACTION_ICONS[type || ''];
  return (
    <>
      <FieldWrapper>
        <DangerZone
          title={OVERVIEW.ACTION_DANGER_ZONE_TITLE}
          subTitle={OVERVIEW.ACTION_DANGER_ZONE_SUBTITLE}
          btnText={OVERVIEW.DELETE}
          onClick={() => setShowModal(true)}
        />
      </FieldWrapper>
      {showModal && (
        <KeyvalModal
          show={showModal}
          closeModal={() => setShowModal(false)}
          config={modalConfig}
        >
          <br />
          <ActionIcon style={{ width: 52, height: 52 }} />
          <br />
          <KeyvalText size={20} weight={600}>
            {`${OVERVIEW.DELETE} ${name}`}
          </KeyvalText>
        </KeyvalModal>
      )}
    </>
  );
}
