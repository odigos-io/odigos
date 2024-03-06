import React from 'react';
import { useRouter } from 'next/navigation';
import { NewActionCard } from '@/components';
import { ActionItemCard, ActionsType } from '@/types';
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
    title: 'Add Cluster Info',
    description: 'Add static cluster-scoped attributes to your data.',
    type: ActionsType.ADD_CLUSTER_INFO,
    icon: ActionsType.ADD_CLUSTER_INFO,
  },
  // {
  //   id: '2',
  //   title: 'Filter',
  //   description: 'Filter spans based on the attributes of the span.',
  //   type: 'filter',
  //   icon: 'filter',
  // },
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
