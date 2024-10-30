import React from 'react';
import { useRouter } from 'next/navigation';
import { NewActionCard } from '@/components';
import { KeyvalLink, KeyvalText } from '@/design.system';
import { ActionItemCard, RulesType } from '@/types';
import { ACTION, INSTRUMENTATION_RULES_DOCS_LINK, OVERVIEW } from '@/utils';
import { ActionCardWrapper, ActionsListWrapper, DescriptionWrapper, LinkWrapper } from './styled';

const ITEMS = [
  {
    id: 'payload-collection',
    title: 'Payload Collection',
    description: 'Record operation payloads as span attributes where supported.',
    type: RulesType.PAYLOAD_COLLECTION,
    icon: RulesType.PAYLOAD_COLLECTION,
  },
];

export function ChooseInstrumentationRuleContainer(): React.JSX.Element {
  const router = useRouter();

  function onItemClick({ item }: { item: ActionItemCard }) {
    router.push(`/create-rule?type=${item.type}`);
  }

  function renderActionsList() {
    return ITEMS.map((item) => {
      return (
        <ActionCardWrapper data-cy={'choose-instrumentation-rule-' + item.type} key={item.id}>
          <NewActionCard item={item} onClick={onItemClick} />
        </ActionCardWrapper>
      );
    });
  }

  return (
    <>
      <DescriptionWrapper>
        <KeyvalText size={14}>{OVERVIEW.INSTRUMENTATION_RULE_DESCRIPTION}</KeyvalText>
        <LinkWrapper>
          <KeyvalLink fontSize={14} value={ACTION.LINK_TO_DOCS} onClick={() => window.open(INSTRUMENTATION_RULES_DOCS_LINK, '_blank')} />
        </LinkWrapper>
      </DescriptionWrapper>
      <ActionsListWrapper>{renderActionsList()}</ActionsListWrapper>
    </>
  );
}
