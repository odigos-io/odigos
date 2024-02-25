import React from 'react';
import { OVERVIEW } from '@/utils/constants';
import { EmptyList } from '@/components/lists';
import { AddItemMenu } from '../../add.item.menu';
import { Destination } from '@/types/destinations';
import { ManagedListWrapper } from './destination.list.styled';
import DestinationManagedCard from './destination.managed.card';

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
    console.log('object');
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
