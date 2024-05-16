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
    id: 'add_cluster_info',
    title: 'Add Cluster Info',
    description: 'Add static cluster-scoped attributes to your data.',
    type: ActionsType.ADD_CLUSTER_INFO,
    icon: ActionsType.ADD_CLUSTER_INFO,
  },
  {
    id: 'delete_attribute',
    title: 'Delete Attribute',
    description: 'Delete attributes from logs, metrics, and traces.',
    type: ActionsType.DELETE_ATTRIBUTES,
    icon: ActionsType.DELETE_ATTRIBUTES,
  },
  {
    id: 'rename_attribute',
    title: 'Rename Attribute',
    description: 'Rename attributes in logs, metrics, and traces.',
    type: ActionsType.RENAME_ATTRIBUTES,
    icon: ActionsType.RENAME_ATTRIBUTES,
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
