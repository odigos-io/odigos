import React from 'react';
import { ManagedListWrapper, ManagedContainer } from './sources.manage.styled';
import SourceManagedCard from './sources.manage.card';
import { ManagedSource } from '@/types/sources';
import { KeyvalText } from '@/design.system';
import { OVERVIEW, ROUTES } from '@/utils/constants';
import { useRouter } from 'next/navigation';
import { EmptyList } from '@/components/lists';

interface SourcesManagedListProps {
  data: ManagedSource[];
  onAddClick: () => void;
}
export function SourcesManagedList({
  data = [],
  onAddClick,
}: SourcesManagedListProps) {
  const router = useRouter();
  function renderSources() {
    return data.map((source: ManagedSource) => (
      <SourceManagedCard
        key={source?.name}
        item={source}
        onClick={() =>
          router.push(
            `${ROUTES.MANAGE_SOURCE}?name=${source?.name}&kind=${source?.kind}&namespace=${source?.namespace}`
          )
        }
      />
    ));
  }

  return data.length === 0 ? (
    <EmptyList
      title={OVERVIEW.EMPTY_SOURCE}
      btnTitle={OVERVIEW.ADD_NEW_SOURCE}
      buttonAction={onAddClick}
    />
  ) : (
    <ManagedContainer>
      <KeyvalText>{`${data.length} ${OVERVIEW.MENU.SOURCES}`}</KeyvalText>
      <br />
      <ManagedListWrapper>{renderSources()}</ManagedListWrapper>
    </ManagedContainer>
  );
}
