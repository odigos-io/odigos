import React from 'react';
import styled from 'styled-components';

interface Props {
  label: string | number | React.ReactNode;
  filled?: boolean;
}

const Styled = styled.span<{ $filled: Props['filled'] }>`
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 100%;
  border: 1px solid ${({ theme, $filled }) => ($filled ? theme.colors.majestic_blue : theme.colors.border)};
  background-color: ${({ theme, $filled }) => ($filled ? theme.colors.majestic_blue : theme.colors.blank_background)};
  color: ${({ theme, $filled }) => ($filled ? theme.colors.secondary : theme.text.grey)};
  font-family: ${({ theme }) => theme.font_family.secondary};
  font-size: 12px;
`;

export const Badge = ({ label, filled }: Props) => {
  return <Styled $filled={filled}>{label}</Styled>;
};
