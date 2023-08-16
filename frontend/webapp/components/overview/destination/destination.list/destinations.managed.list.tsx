import React from 'react';
import DestinationManagedCard from './destination.managed.card';
import { KeyvalText, KeyvalButton } from '@/design.system';
import { Plus } from '@/assets/icons/overview';
import { OVERVIEW } from '@/utils/constants';
import theme from '@/styles/palette';
import { MenuWrapper, ManagedListWrapper } from './destination.list.styled';
import { Destination } from '@/types/destinations';
import { EmptyList } from '@/components/lists';

const BUTTON_STYLES = { gap: 10, width: 224, height: 40 };
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
        key={destination?.id}
        item={destination}
      />
    ));
  }

  return (
    <>
      {data?.length === 0 ? (
        <EmptyList
          title={OVERVIEW.EMPTY_DESTINATION}
          btnTitle={OVERVIEW.ADD_NEW_DESTINATION}
          btnAction={onMenuButtonClick}
        />
      ) : (
        <>
          <MenuWrapper>
            <KeyvalText>{`${data.length} ${OVERVIEW.MENU.DESTINATIONS}`}</KeyvalText>
            <KeyvalButton onClick={onMenuButtonClick} style={BUTTON_STYLES}>
              <Plus />
              <KeyvalText size={16} weight={700} color={theme.text.dark_button}>
                {OVERVIEW.ADD_NEW_DESTINATION}
              </KeyvalText>
            </KeyvalButton>
          </MenuWrapper>
          <ManagedListWrapper>{renderDestinations()}</ManagedListWrapper>
        </>
      )}
    </>
  );
}
