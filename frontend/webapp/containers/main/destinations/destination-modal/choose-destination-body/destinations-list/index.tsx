import React from 'react';
import styled from 'styled-components';
import { DestinationTypeItem } from '@/types';
import { IDestinationListItem } from '@/hooks';
import { capitalizeFirstLetter } from '@/utils';
import { DestinationListItem } from './destination-list-item';
import { NoDataFound, SectionTitle } from '@/reuseable-components';
import { PotentialDestinationsList } from './potential-destinations-list';

const Container = styled.div`
  display: flex;
  flex-direction: column;
  align-self: stretch;
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

const DestinationsList: React.FC<DestinationsListProps> = ({ items, setSelectedItems }) => {
  function renderCategoriesList() {
    if (!items.length) {
      return (
        <NoDataFoundWrapper>
          <NoDataFound title='No destinations found' />
        </NoDataFoundWrapper>
      );
    }

    return items.map((item) => {
      return (
        <ListsWrapper key={`category-${item.name}`}>
          <SectionTitle size='small' title={capitalizeFirstLetter(item.name)} description={item.description} />
          {item.items.map((categoryItem) => (
            <DestinationListItem key={`destination-${categoryItem.type}`} item={categoryItem} onSelect={setSelectedItems} />
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

export { DestinationsList };
