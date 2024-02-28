import React from 'react';
import theme from '@/styles/palette';
import { ActionData } from '@/types';
import styled from 'styled-components';
import { ACTION_ICONS } from '@/assets';
import { KeyvalCard, KeyvalTag, KeyvalText } from '@/design.system';

interface NewActionCardProps {
  item: ActionData;
  onClick: ({ item }: { item: ActionData }) => void;
}

const ACTION_TYPE_TEXT = {
  AddClusterInfo: 'Add Cluster Info',
  filter: 'Filter',
};

const CardContentWrapper = styled.div`
  padding: 12px;
  display: flex;
  justify-content: center;
  align-items: center;
  flex-direction: column;
  width: 208px;
  height: 100%;
  gap: 8px;
  border: 1px solid transparent;
  cursor: pointer;
  &:hover {
    border-radius: 24px;
    border: 1px solid ${theme.colors.secondary};
  }
`;

export function ManagedActionCard({ item, onClick }: NewActionCardProps) {
  const SvgIcon = ACTION_ICONS[item.type];

  //TODO: render card content based on the item type

  return (
    <KeyvalCard>
      <CardContentWrapper onClick={() => onClick({ item })}>
        <SvgIcon style={{ width: 32, height: 32 }} />
        <KeyvalText size={18} weight={700}>
          {item.spec.actionName}
        </KeyvalText>
        <KeyvalTag title={ACTION_TYPE_TEXT[item.type]} />
        <KeyvalText color={theme.text.light_grey} size={14} weight={400}>
          {`${item?.spec.clusterAttributes.length} cluster attributes`}
        </KeyvalText>
      </CardContentWrapper>
    </KeyvalCard>
  );
}
