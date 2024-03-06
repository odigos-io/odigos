'use client';
import React, { useState } from 'react';
import { styled } from 'styled-components';
import { ConnectionsIcons } from '@/components/setup';
import { DangerZone, KeyvalModal, KeyvalText } from '@/design.system';
import { ModalPositionX, ModalPositionY } from '@/design.system/modal/types';
import { OVERVIEW } from '@/utils/constants';

const FieldWrapper = styled.div`
  margin-top: 32px;
  width: 348px;
`;

const IMAGE_STYLE = { border: 'solid 1px #ededed' };
export default function FormDangerZone({
  onDelete,
  data,
}: {
  onDelete: () => void;
  data: any;
}) {
  const [showModal, setShowModal] = useState(false);

  const modalConfig = {
    title: OVERVIEW.DELETE_DESTINATION,
    showHeader: true,
    showOverlay: true,
    positionX: ModalPositionX.center,
    positionY: ModalPositionY.center,
    padding: '20px',
    footer: {
      primaryBtnText: OVERVIEW.DELETE_BUTTON,
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
          title={OVERVIEW.DELETE_MODAL_TITLE}
          subTitle={OVERVIEW.DELETE_MODAL_SUBTITLE}
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
          <ConnectionsIcons icon={data?.image_url} imageStyle={IMAGE_STYLE} />
          <br />
          <KeyvalText size={20} weight={600}>
            {`${OVERVIEW.DELETE} ${data?.name}`}
          </KeyvalText>
        </KeyvalModal>
      )}
    </>
  );
}
