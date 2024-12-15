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
            <DataTab
              key={`destination-${categoryItem.type}`}
              title={categoryItem.displayName}
              logo={categoryItem.imageUrl}
              hoverText='Select'
              monitors={Object.keys(categoryItem.supportedSignals).filter((signal) => categoryItem.supportedSignals[signal].supported)}
              monitorsWithLabels
              onClick={() => setSelectedItems(categoryItem)}
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

export { DestinationsList };
