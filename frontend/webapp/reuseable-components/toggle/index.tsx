import Image from 'next/image';
import { Text } from '../text';
import { Tooltip } from '../tooltip';
import styled from 'styled-components';
import React, { useEffect, useState } from 'react';

interface ToggleProps {
  title: string;
  tooltip?: string;
  initialValue?: boolean;
  onChange?: (value: boolean) => void;
  disabled?: boolean;
}

const Container = styled.div<{ $disabled?: ToggleProps['disabled'] }>`
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: ${({ $disabled }) => ($disabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ $disabled }) => ($disabled ? 0.6 : 1)};
`;

const ToggleSwitch = styled.div<{ $isActive: boolean; $disabled?: ToggleProps['disabled'] }>`
  width: 24px;
  height: 12px;
  border: 1px ${({ $isActive, theme }) => ($isActive ? `solid ${theme.colors.majestic_blue}` : 'dashed #aaa')};
  border-radius: 20px;
  display: flex;
  align-items: center;
  padding: 2px;
  background-color: transparent;
  pointer-events: ${({ $disabled }) => ($disabled ? 'none' : 'auto')};
  cursor: ${({ $disabled }) => ($disabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ $isActive }) => ($isActive ? 1 : 0.5)};
  transition: border-color 0.3s, opacity 0.3s;
  &::before {
    content: '';
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background-color: ${({ $isActive, theme }) => ($isActive ? theme.colors.majestic_blue : theme.colors.secondary)};
    transform: ${({ $isActive }) => ($isActive ? 'translateX(12px)' : 'translateX(0)')};
    transition: background-color 0.3s, transform 0.3s;
  }
`;

const Toggle: React.FC<ToggleProps> = ({ title, tooltip, initialValue = false, onChange, disabled }) => {
  const [isActive, setIsActive] = useState(initialValue);
  useEffect(() => setIsActive(initialValue), [initialValue]);

  const handleToggle: React.MouseEventHandler<HTMLDivElement> = (e) => {
    if (disabled) return;

    e.stopPropagation();

    setIsActive((prev) => {
      const newValue = !prev;
      if (onChange) onChange(newValue);
      return newValue;
    });
  };

  return (
    <Container $disabled={disabled} onClick={handleToggle}>
      <ToggleSwitch $disabled={disabled} $isActive={isActive} />
      <Tooltip text={tooltip} withIcon>
        <Text size={14}>{title}</Text>
      </Tooltip>
    </Container>
  );
};

export { Toggle };
