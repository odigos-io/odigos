import React from 'react';
import styled from 'styled-components';
import { ActionItemCard } from '@/types';
import { KeyvalCard, KeyvalText } from '@/design.system';
import theme from '@/styles/palette';
import { ACTION_ICONS } from '@/assets';

interface NewActionCardProps {
  item: ActionItemCard;
}

const CardContentWrapper = styled.div`
  padding: 12px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  max-width: 220px;
  border: 1px solid transparent;
  cursor: pointer;
  &:hover {
    border-radius: 24px;
    border: 1px solid ${theme.colors.secondary};
  }
`;
export function NewActionCard({ item }: NewActionCardProps) {
  const SvgIcon = ACTION_ICONS[item.icon];

  return (
    <KeyvalCard>
      <CardContentWrapper>
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
