import React from "react";
import DestinationManagedCard from "./destination.managed.card";
import { styled } from "styled-components";
import { KeyvalText, KeyvalButton } from "@/design.system";
import { Plus } from "@/assets/icons/overview";

export const ManagedListWrapper = styled.div`
  width: 100%;
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  overflow-y: scroll;
  padding: 0px 36px;
`;

export const MenuWrapper = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 32px 36px;
`;

export default function DestinationsManagedList({
  data,
  onItemClick,
  onMenuButtonClick,
}: any) {
  function renderDestinations() {
    return data.map((destination: any) => (
      <DestinationManagedCard
        onClick={() => onItemClick(destination)}
        key={destination?.name}
        item={destination}
      />
    ));
  }

  return (
    <>
      <MenuWrapper>
        <KeyvalText>{`${data.length} Applications`}</KeyvalText>
        <KeyvalButton
          onClick={onMenuButtonClick}
          style={{ gap: 10, width: 224, height: 40 }}
        >
          <Plus />
          <KeyvalText size={16} weight={700} color="#0A1824">
            {"Add New Destination"}
          </KeyvalText>
        </KeyvalButton>
      </MenuWrapper>
      <ManagedListWrapper>{renderDestinations()}</ManagedListWrapper>{" "}
    </>
  );
}
