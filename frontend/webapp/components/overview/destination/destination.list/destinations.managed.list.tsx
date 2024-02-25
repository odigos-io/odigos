import React from 'react';
import theme from '@/styles/palette';
import { OVERVIEW } from '@/utils/constants';
import { Plus } from '@/assets/icons/overview';
import { EmptyList } from '@/components/lists';
import { AddItemMenu } from '../../add.item.menu';
import { Destination } from '@/types/destinations';
import { KeyvalText, KeyvalButton } from '@/design.system';
import DestinationManagedCard from './destination.managed.card';
import { MenuWrapper, ManagedListWrapper } from './destination.list.styled';

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
          <AddItemMenu
            length={data?.length}
            onClick={onMenuButtonClick}
            btnLabel={OVERVIEW.ADD_NEW_DESTINATION}
            lengthLabel={OVERVIEW.MENU.DESTINATIONS}
          />
          <ManagedListWrapper>{renderDestinations()}</ManagedListWrapper>
        </>
      )}
    </>
  );
}
