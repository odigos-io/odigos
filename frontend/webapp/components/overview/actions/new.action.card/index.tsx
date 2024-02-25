import React from 'react';
import theme from '@/styles/palette';
import styled from 'styled-components';
import { ACTION_ICONS } from '@/assets';
import { ActionItemCard } from '@/types';
import { KeyvalCard, KeyvalText } from '@/design.system';

interface NewActionCardProps {
  item: ActionItemCard;
  onClick: () => void;
}

const CardContentWrapper = styled.div`
  padding: 12px;
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 8px;
  border: 1px solid transparent;
  cursor: pointer;
  &:hover {
    border-radius: 24px;
    border: 1px solid ${theme.colors.secondary};
  }
`;

export function NewActionCard({ item, onClick }: NewActionCardProps) {
  const SvgIcon = ACTION_ICONS[item.icon];

  return (
    <KeyvalCard>
      <CardContentWrapper onClick={onClick}>
        <SvgIcon style={{ width: 56, height: 56 }} />
        <KeyvalText size={18} weight={700}>
          {item.title}
        </KeyvalText>
        <KeyvalText color={theme.text.light_grey} size={14} weight={400}>
          {item.description}
        </KeyvalText>
      </CardContentWrapper>
    </KeyvalCard>
  );
}
