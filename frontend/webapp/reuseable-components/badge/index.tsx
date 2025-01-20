import React from 'react';
import styled from 'styled-components';

interface Props {
  label: string | number | React.ReactNode;
  filled?: boolean;
}

const Styled = styled.span<{ $filled: Props['filled'] }>`
  min-width: 20px;
  height: 20px;
  padding: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50px;
  border: 1px solid ${({ theme, $filled }) => ($filled ? theme.colors.majestic_blue : theme.colors.border)};
  background-color: ${({ theme, $filled }) => ($filled ? theme.colors.majestic_blue : theme.colors.blank_background)};
  color: ${({ theme, $filled }) => ($filled ? theme.colors.secondary : theme.text.grey)};
  font-family: ${({ theme }) => theme.font_family.secondary};
  font-size: 12px;
`;

export const Badge = ({ label, filled }: Props) => {
  return <Styled $filled={filled}>{label}</Styled>;
};
