import { SVG } from '@/assets';
import { FlexRow } from '@/styles';
import React, { CSSProperties } from 'react';
import styled from 'styled-components';
import { Text } from '../text';

type SelectedValue = any;

interface Props {
  options: {
    icon?: SVG;
    label?: string;
    value: SelectedValue;
    selectedBgColor?: CSSProperties['backgroundColor'];
  }[];
  selected: SelectedValue;
  setSelected: (value: SelectedValue) => void;
}

const Container = styled(FlexRow)`
  gap: 0;
`;

const Button = styled.button<{ $selected: boolean; $isFirstItem: boolean; $isLastItem: boolean; $bgColor?: CSSProperties['backgroundColor'] }>`
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 6px 12px;
  background-color: ${({ theme, $selected, $bgColor }) => ($selected ? $bgColor || theme.colors.white_opacity['008'] : 'transparent')};
  border-radius: ${({ $isFirstItem, $isLastItem }) => ($isFirstItem ? '32px 0px 0px 32px' : $isLastItem ? '0px 32px 32px 0px' : '0')};
  border: 1px solid ${({ theme }) => theme.colors.border};
  cursor: pointer;
  &:hover {
    border: 1px solid ${({ theme }) => theme.colors.secondary};
  }
  transition: background-color 0.3s;
`;

export const Segment: React.FC<Props> = ({ options = [], selected, setSelected }) => {
  return (
    <Container>
      {options.map(({ icon: Icon, label, value, selectedBgColor }, idx) => {
        const isSelected = selected === value;

        return (
          <Button $isFirstItem={idx === 0} $isLastItem={idx === options.length - 1} $bgColor={selectedBgColor} $selected={isSelected} onClick={() => setSelected(value)}>
            {Icon && <Icon />}
            {label && (
              <Text size={12} family='secondary' decoration='underline'>
                {label}
              </Text>
            )}
          </Button>
        );
      })}
    </Container>
  );
};
