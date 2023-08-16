import React from 'react';
import { KeyvalText } from '@/design.system';
import { DestinationCard } from '../destination.card/destination.card';
import {
  DestinationListWrapper,
  DestinationTypeTitleWrapper,
} from './destination.list.styled';
import { capitalizeFirstLetter } from '@/utils/functions';
import { ROUTES } from '@/utils/constants';

export function DestinationList({ data: { items, name }, onItemClick }: any) {
  function renderList() {
    return items?.map((item: any, index: number) => (
      <DestinationCard
        key={index}
        item={item}
        onClick={() => onItemClick(item)}
      />
    ));
  }
  const getNumberOfItemsRepeated = () =>
    window.location.pathname.includes(ROUTES.CREATE_DESTINATION) ? 5 : 4;

  return items?.length ? (
    <>
      <DestinationTypeTitleWrapper>
        <KeyvalText>{`${items?.length} ${capitalizeFirstLetter(
          name
        )}`}</KeyvalText>
      </DestinationTypeTitleWrapper>
      <DestinationListWrapper repeat={getNumberOfItemsRepeated()}>
        {renderList()}
      </DestinationListWrapper>
    </>
  ) : null;
}
