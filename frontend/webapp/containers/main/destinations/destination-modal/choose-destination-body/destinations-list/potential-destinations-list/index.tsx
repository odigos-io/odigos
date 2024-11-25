import React from 'react';
import styled from 'styled-components';
import { DestinationTypeItem } from '@/types';
import { usePotentialDestinations } from '@/hooks';
import { DestinationListItem } from '../destination-list-item';
import { SectionTitle, SkeletonLoader } from '@/reuseable-components';

const ListsWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

interface PotentialDestinationsListProps {
  setSelectedItems: (item: DestinationTypeItem) => void;
}

export const PotentialDestinationsList: React.FC<PotentialDestinationsListProps> = ({ setSelectedItems }) => {
  const { loading, data } = usePotentialDestinations();

  if (!data.length) return null;

  return (
    <ListsWrapper>
      <SectionTitle
        size='small'
        icon='/brand/odigos-icon.svg'
        title='Detected by Odigos'
        description='Odigos detects destinations for which automatic connection is available. All data will be filled out automatically.'
      />
      {loading ? <SkeletonLoader size={1} /> : data.map((item) => <DestinationListItem key={item.displayName} item={item} onSelect={setSelectedItems} />)}
    </ListsWrapper>
  );
};
