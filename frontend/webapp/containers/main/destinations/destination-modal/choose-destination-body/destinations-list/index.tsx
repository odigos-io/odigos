import React from 'react';
import styled from 'styled-components';
import { DestinationTypeItem } from '@/types';
import { IDestinationListItem } from '@/hooks';
import { capitalizeFirstLetter } from '@/utils';
import { DataTab, NoDataFound, SectionTitle } from '@/reuseable-components';
import { PotentialDestinationsList } from './potential-destinations-list';

const Container = styled.div`
  display: flex;
  flex-direction: column;
  align-self: stretch;
  gap: 24px;
  max-height: calc(100vh - 450px);
  overflow-y: auto;

  @media (height < 800px) {
    max-height: calc(100vh - 400px);
  }
`;

const ListsWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

const NoDataFoundWrapper = styled(Container)`
  margin-top: 80px;
`;

interface DestinationsListProps {
  items: IDestinationListItem[];
  setSelectedItems: (item: DestinationTypeItem) => void;
}

export const DestinationsList: React.FC<DestinationsListProps> = ({ items, setSelectedItems }) => {
  function renderCategoriesList() {
    if (!items.length) {
      return (
        <NoDataFoundWrapper>
          <NoDataFound title='No destinations found' />
        </NoDataFoundWrapper>
      );
    }

    return items.map((categoryItem) => {
      return (
        <ListsWrapper key={`category-${categoryItem.name}`}>
          <SectionTitle size='small' title={capitalizeFirstLetter(categoryItem.name)} description={categoryItem.description} />
          {categoryItem.items.map((destinationItem) => (
            <DataTab
              key={`destination-${destinationItem.type}`}
              data-id={`destination-${destinationItem.displayName}`}
              title={destinationItem.displayName}
              iconSrc={destinationItem.imageUrl}
              hoverText='Select'
              monitors={Object.keys(destinationItem.supportedSignals || {}).filter((signal) => destinationItem.supportedSignals?.[signal]?.supported)}
              monitorsWithLabels
              onClick={() => setSelectedItems(destinationItem)}
            />
          ))}
        </ListsWrapper>
      );
    });
  }

  return (
    <Container>
      <PotentialDestinationsList setSelectedItems={setSelectedItems} />
      {renderCategoriesList()}
    </Container>
  );
};
