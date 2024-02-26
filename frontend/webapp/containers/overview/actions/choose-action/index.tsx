import React from 'react';
import { ActionItemCard } from '@/types';
import { NewActionCard } from '@/components';
import { KeyvalLink, KeyvalText } from '@/design.system';
import { ACTION, ACTION_DOCS_LINK, OVERVIEW } from '@/utils';
import {
  LinkWrapper,
  ActionCardWrapper,
  ActionsListWrapper,
  DescriptionWrapper,
} from './styled';
import { useRouter } from 'next/navigation';

const ITEMS = [
  {
    id: '1',
    title: 'Insert Cluster Attributes',
    description:
      'With cluster attributes, you can define the attributes of the cluster. This is useful for filtering and grouping spans in your backend.',
    type: 'insert-cluster-attributes',
    icon: 'insert-cluster-attributes',
  },
  {
    id: '2',
    title: 'Filter',
    description: 'Filter spans based on the attributes of the span.',
    type: 'filter',
    icon: 'filter',
  },
];

export function ChooseActionContainer(): React.JSX.Element {
  const router = useRouter();

  function onItemClick({ item }: { item: ActionItemCard }) {
    router.push(`/create-action?type=${item.type}`);
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
        <KeyvalText size={14}>{OVERVIEW.ACTION_DESCRIPTION}</KeyvalText>
        <LinkWrapper>
          <KeyvalLink
            fontSize={14}
            value={ACTION.LINK_TO_DOCS}
            onClick={() => window.open(ACTION_DOCS_LINK, '_blank')}
          />
        </LinkWrapper>
      </DescriptionWrapper>
      <ActionsListWrapper>{renderActionsList()}</ActionsListWrapper>
    </>
  );
}
