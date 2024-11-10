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

const Container = styled.div<{ disabled?: boolean }>`
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ disabled }) => (disabled ? 0.6 : 1)};
`;

const ToggleSwitch = styled.div<{ isActive: boolean; disabled?: boolean }>`
  width: 24px;
  height: 12px;
  border: 1px dashed #aaa;
  border-radius: 20px;
  display: flex;
  align-items: center;
  padding: 2px;
  background-color: ${({ isActive, theme }) => (isActive ? theme.colors.primary : 'transparent')};
  pointer-events: ${({ disabled }) => (disabled ? 'none' : 'auto')};
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ isActive }) => (isActive ? 1 : 0.4)};
  transition: background-color 0.3s, opacity 0.3s;
  &::before {
    content: '';
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background-color: ${({ theme }) => theme.colors.secondary};
    transform: ${({ isActive }) => (isActive ? 'translateX(12px)' : 'translateX(0)')};
    transition: transform 0.3s;
  }
`;

const Toggle: React.FC<ToggleProps> = ({ title, tooltip, initialValue = false, onChange, disabled }) => {
  const [isActive, setIsActive] = useState(initialValue);
  useEffect(() => setIsActive(initialValue), [initialValue]);

  const handleToggle = () => {
    if (disabled) return;

    let newValue = initialValue;

    setIsActive((prev) => {
      newValue = !prev;
      return newValue;
    });

    if (onChange) onChange(newValue);
  };

  return (
    <Tooltip text={tooltip || ''}>
      <Container disabled={disabled} onClick={handleToggle}>
        <ToggleSwitch isActive={isActive} disabled={disabled} />
        <Text size={14}>{title}</Text>
      </Container>

      {tooltip && <Image src='/icons/common/info.svg' alt='' width={16} height={16} />}
    </Tooltip>
  );
};

export { Toggle };
