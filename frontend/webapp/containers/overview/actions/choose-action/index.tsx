import React from 'react';
import styled from 'styled-components';
import { NewActionCard } from '@/components';
import { KeyvalLink, KeyvalText } from '@/design.system';

const ACTION_DOCS_LINK = 'https://docs.odigos.io/pipeline/actions/introduction';

const ITEMS = [
  {
    id: '1',
    title: 'Cluster Attributes',
    description:
      'With cluster attributes, you can define the attributes of the cluster. This is useful for filtering and grouping spans in your backend.',
    type: 'cluster-attributes',
    icon: 'cluster-attributes',
  },
  {
    id: '2',
    title: 'Filter',
    description: 'Filter spans based on the attributes of the span.',
    type: 'filter',
    icon: 'filter',
  },
];

const ActionsListWrapper = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
  padding: 0 24px 24px 24px;
  overflow-y: auto;
  align-items: start;
  max-height: 100%;
  padding-bottom: 220px;
  box-sizing: border-box;
`;

const DescriptionWrapper = styled.div`
  padding: 24px;
  gap: 4px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
`;

const LinkWrapper = styled.div`
  width: 100px;
`;

export function ChooseActionContainer() {
  function onItemClick() {
    console.log('Item clicked');
  }

  function renderActionsList() {
    return ITEMS.map((item) => {
      return (
        <div style={{ height: '100%', maxHeight: 220 }} key={item.id}>
          <NewActionCard item={item} onClick={onItemClick} />
        </div>
      );
    });
  }

  return (
    <>
      <DescriptionWrapper>
        <KeyvalText>
          {
            'Actions are a way to modify the OpenTelemetry data recorded by Odigos Sources, before it is exported to your Odigos Destinations.'
          }
        </KeyvalText>
        <LinkWrapper>
          <KeyvalLink
            value="Learn more"
            onClick={() => window.open(ACTION_DOCS_LINK, '_blank')}
          />
        </LinkWrapper>
      </DescriptionWrapper>
      <ActionsListWrapper>{renderActionsList()}</ActionsListWrapper>
    </>
  );
}
