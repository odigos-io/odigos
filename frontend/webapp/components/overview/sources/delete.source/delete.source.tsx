import React, { useState } from "react";
import { styled } from "styled-components";
import { ConnectionsIcons } from "@/components/setup";
import { DangerZone, KeyvalModal, KeyvalText } from "@/design.system";
import { ModalPositionX, ModalPositionY } from "@/design.system/modal/types";
import theme from "@/styles/palette";
import { OVERVIEW } from "@/utils/constants";

const FieldWrapper = styled.div`
  margin-top: 32px;
  width: 348px;
`;

const IMAGE_STYLE = { border: "solid 1px #ededed" };
export function DeleteSource({
  onDelete,
  name,
  image_url,
}: {
  onDelete: () => void;
  name: string | undefined;
  image_url: string;
}) {
  const [showModal, setShowModal] = useState(false);

  const modalConfig = {
    title: OVERVIEW.DELETE_SOURCE,
    showHeader: true,
    showOverlay: true,
    positionX: ModalPositionX.center,
    positionY: ModalPositionY.center,
    padding: "20px",
    footer: {
      primaryBtnText: OVERVIEW.CONFIRM_SOURCE_DELETE,
      primaryBtnAction: () => {
        setShowModal(false);
      },
    },
  };

  return (
    <>
      <FieldWrapper>
        <DangerZone
          title={OVERVIEW.SOURCE_DANGER_ZONE_TITLE}
          subTitle={OVERVIEW.SOURCE_DANGER_ZONE_SUBTITLE}
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
          <ConnectionsIcons icon={image_url} imageStyle={IMAGE_STYLE} />
          <br />
          <KeyvalText color={theme.text.primary} size={20} weight={600}>
            {`${OVERVIEW.DELETE} ${name}`}
          </KeyvalText>
        </KeyvalModal>
      )}
    </>
  );
}
