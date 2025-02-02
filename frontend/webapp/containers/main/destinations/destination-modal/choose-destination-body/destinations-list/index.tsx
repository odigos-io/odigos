import React from 'react';
import styled from 'styled-components';
import { IDestinationListItem } from '@/hooks';
import type { DestinationTypeItem } from '@/types';
import { PotentialDestinationsList } from './potential-destinations-list';
import { capitalizeFirstLetter, DataTab, NoDataFound, SectionTitle, SIGNAL_TYPE } from '@odigos/ui-components';

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
          {categoryItem.items.map((item, idx) => (
            <DataTab
              key={`select-destination-${item.type}-${idx}`}
              data-id={`select-destination-${item.type}`}
              title={item.displayName}
              iconSrc={item.imageUrl}
              hoverText='Select'
              monitors={Object.keys(item.supportedSignals).filter((signal: SIGNAL_TYPE) => item.supportedSignals[signal].supported) as SIGNAL_TYPE[]}
              monitorsWithLabels
              onClick={() => setSelectedItems(item)}
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
