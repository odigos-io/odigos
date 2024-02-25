import React from 'react';
import { NewActionCard } from '@/components';
import { KeyvalLink, KeyvalText } from '@/design.system';
import { ACTION, ACTION_DOCS_LINK, OVERVIEW } from '@/utils';
import {
  LinkWrapper,
  ActionCardWrapper,
  ActionsListWrapper,
  DescriptionWrapper,
} from './styled';

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

export function ChooseActionContainer() {
  function onItemClick() {
    console.log('Item clicked');
  }

  function renderActionsList() {
    return ITEMS.map((item) => {
      return (
        <ActionCardWrapper key={item.id}>
          <NewActionCard item={item} onClick={onItemClick} />
        </ActionCardWrapper>
      );
    });
  }

  return (
    <>
      <DescriptionWrapper>
        <KeyvalText>{OVERVIEW.ACTION_DESCRIPTION}</KeyvalText>
        <LinkWrapper>
          <KeyvalLink
            value={ACTION.LEARN_MORE}
            onClick={() => window.open(ACTION_DOCS_LINK, '_blank')}
          />
        </LinkWrapper>
      </DescriptionWrapper>
      <ActionsListWrapper>{renderActionsList()}</ActionsListWrapper>
    </>
  );
}
