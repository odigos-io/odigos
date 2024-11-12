import Image from 'next/image';
import { Text } from '../text';
import { Tooltip } from '../tooltip';
import styled from 'styled-components';
import React, { useEffect, useState } from 'react';

interface ToggleProps {
  title: string;
  tooltip?: string;
  initialValue?: boolean;
<<<<<<< HEAD
  onChange?: (value: boolean) => void;
=======
  onChange: (value: boolean) => void;
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
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
<<<<<<< HEAD
  border: 1px dashed #aaa;
=======
  border: 1px ${({ isActive, theme }) => (isActive ? `solid ${theme.colors.majestic_blue}` : 'dashed #aaa')};
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  border-radius: 20px;
  display: flex;
  align-items: center;
  padding: 2px;
<<<<<<< HEAD
  background-color: ${({ isActive, theme }) => (isActive ? theme.colors.primary : 'transparent')};
  pointer-events: ${({ disabled }) => (disabled ? 'none' : 'auto')};
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ isActive }) => (isActive ? 1 : 0.4)};
  transition: background-color 0.3s, opacity 0.3s;
=======
  background-color: transparent;
  pointer-events: ${({ disabled }) => (disabled ? 'none' : 'auto')};
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ isActive }) => (isActive ? 1 : 0.5)};
  transition: border-color 0.3s, opacity 0.3s;
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  &::before {
    content: '';
    width: 12px;
    height: 12px;
    border-radius: 50%;
<<<<<<< HEAD
    background-color: ${({ theme }) => theme.colors.secondary};
    transform: ${({ isActive }) => (isActive ? 'translateX(12px)' : 'translateX(0)')};
    transition: transform 0.3s;
=======
    background-color: ${({ isActive, theme }) => (isActive ? theme.colors.majestic_blue : theme.colors.secondary)};
    transform: ${({ isActive }) => (isActive ? 'translateX(12px)' : 'translateX(0)')};
    transition: background-color 0.3s, transform 0.3s;
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  }
`;

const Toggle: React.FC<ToggleProps> = ({ title, tooltip, initialValue = false, onChange, disabled }) => {
  const [isActive, setIsActive] = useState(initialValue);
<<<<<<< HEAD
  useEffect(() => setIsActive(initialValue), [initialValue]);

  const handleToggle = () => {
    if (disabled) return;

    let newValue = initialValue;

    setIsActive((prev) => {
      newValue = !prev;
      return newValue;
    });

    if (onChange) onChange(newValue);
=======
  useEffect(() => onChange(isActive), [isActive]);

  const handleToggle = () => {
    if (disabled) return;
    setIsActive((prev) => !prev);
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
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
