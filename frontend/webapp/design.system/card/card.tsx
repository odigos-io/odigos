import React from 'react';
import { Card } from '@keyval-dev/design-system';
import styled from 'styled-components';
interface CardProps {
  children: JSX.Element | JSX.Element[];
  focus?: any;
}

interface CardContainerProps {
  selected?: any;
}

export const CardContainer = styled.div<CardContainerProps>`
  display: inline-flex;
  position: relative;
  flex-direction: column;
  border-radius: 24px;
  border: ${({ selected, theme }) =>
    `1px solid ${selected ? theme.colors.secondary : theme.colors.dark_blue}`};
  background: ${({ theme }) => theme.colors.dark};
`;
export function KeyvalCard(props: CardProps) {
  return <CardContainer {...props}>{props.children}</CardContainer>;
}
