import React, { CSSProperties, useEffect, useRef, useState } from 'react';
import { SVG } from '@/assets';
import { Text } from '../text';
import { FlexRow } from '@/styles';
import styled from 'styled-components';

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
  position: relative;
  gap: 0;
`;

const Button = styled.button<{ $isFirstItem: boolean; $isLastItem: boolean }>`
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 6px 12px;
  background-color: transparent;
  border-radius: ${({ $isFirstItem, $isLastItem }) => ($isFirstItem ? '32px 0px 0px 32px' : $isLastItem ? '0px 32px 32px 0px' : '0')};
  border: 1px solid ${({ theme }) => theme.colors.border};
  cursor: pointer;
  &:hover {
    border: 1px solid ${({ theme }) => theme.colors.secondary};
  }
`;

const Background = styled.div<{ $bgColor?: CSSProperties['backgroundColor']; $width: number; $height: number; $x: number; $y: number; $isFirstItem: boolean; $isLastItem: boolean }>`
  position: absolute;
  top: ${({ $y }) => $y}px;
  left: ${({ $x }) => $x}px;
  z-index: -1;
  width: ${({ $width }) => $width}px;
  height: ${({ $height }) => $height}px;
  background-color: ${({ theme, $bgColor }) => $bgColor || theme.colors.white_opacity['008']};
  border-radius: ${({ $isFirstItem, $isLastItem }) => ($isFirstItem ? '32px 0px 0px 32px' : $isLastItem ? '0px 32px 32px 0px' : '0')};
  transition: all 0.3s;
`;

export const Segment: React.FC<Props> = ({ options = [], selected, setSelected }) => {
  const selectedIdx = options.findIndex((option) => option.value === selected);
  const [bgColor, setBgColor] = useState(options[selectedIdx]?.selectedBgColor || '');
  const [bgSize, setBgSize] = useState({ width: 0, height: 0 });
  const [bgPosition, setBgPosition] = useState({ x: 0, y: 0 });
  const selectedRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!!selectedRef.current) {
      setBgSize({
        width: selectedRef.current.offsetWidth,
        height: selectedRef.current.offsetHeight,
      });
      setBgPosition({
        x: selectedRef.current.offsetWidth * selectedIdx,
        y: 0,
      });
    }
  }, [selected, selectedIdx]);

  return (
    <Container>
      {options.map(({ icon: Icon, label, value, selectedBgColor }, idx) => {
        const isSelected = selected === value;

        return (
          <Button
            ref={isSelected ? selectedRef : undefined}
            $isFirstItem={idx === 0}
            $isLastItem={idx === options.length - 1}
            onClick={() => {
              setSelected(value);
              setBgColor(selectedBgColor || '');
            }}
          >
            {Icon && <Icon />}
            {label && (
              <Text size={12} family='secondary' decoration='underline'>
                {label}
              </Text>
            )}
          </Button>
        );
      })}

      <Background
        $bgColor={bgColor}
        $width={bgSize.width}
        $height={bgSize.height}
        $x={bgPosition.x}
        $y={bgPosition.y}
        $isFirstItem={selectedIdx === 0}
        $isLastItem={selectedIdx === options.length - 1}
      />
    </Container>
  );
};
