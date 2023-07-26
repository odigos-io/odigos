import React from "react";
import DestinationManagedCard from "./destination.managed.card";
import { KeyvalText, KeyvalButton } from "@/design.system";
import { Plus } from "@/assets/icons/overview";
import { OVERVIEW, SETUP } from "@/utils/constants";
import theme from "@/styles/palette";
import { MenuWrapper, ManagedListWrapper } from "./destination.list.styled";
import { Destination } from "@/types/destinations";

interface DestinationsManagedListProps {
  data: Destination[];
  onItemClick: (destination: Destination) => void;
  onMenuButtonClick: () => void;
}

export function DestinationsManagedList({
  data,
  onItemClick,
  onMenuButtonClick,
}: DestinationsManagedListProps) {
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
        <KeyvalText>{`${data.length} ${SETUP.APPLICATIONS}`}</KeyvalText>
        <KeyvalButton
          onClick={onMenuButtonClick}
          style={{ gap: 10, width: 224, height: 40 }}
        >
          <Plus />
          <KeyvalText size={16} weight={700} color={theme.text.dark_button}>
            {OVERVIEW.ADD_NEW_DESTINATION}
          </KeyvalText>
        </KeyvalButton>
      </MenuWrapper>
      <ManagedListWrapper>{renderDestinations()}</ManagedListWrapper>
    </>
  );
}
