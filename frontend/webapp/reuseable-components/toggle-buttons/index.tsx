import Image from 'next/image';
import { Tooltip } from '../tooltip';
import styled from 'styled-components';
import React, { useEffect, useState } from 'react';

interface ToggleProps {
  activeText?: string;
  inactiveText?: string;
  tooltip?: string;
  initialValue?: boolean;
  onChange?: (value: boolean) => void;
  disabled?: boolean;
}

const Container = styled.div`
  width: 100%;
  display: flex;
  align-items: center;
`;

const BaseButton = styled.button`
  width: 100%;
  padding: 12px;
  gap: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid ${({ theme }) => theme.colors.border};
  color: ${({ theme }) => theme.colors.secondary};
  font-family: ${({ theme }) => theme.font_family.secondary};
  font-size: 14px;
  text-decoration: underline;
  text-transform: uppercase;
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ disabled }) => (disabled ? 0.6 : 1)};
`;

const ActiveButton = styled(BaseButton)`
  border-radius: 32px 0 0 32px;
  background-color: ${({ theme }) => theme.colors.blank_background};
  &.colored {
    background-color: ${({ theme }) => theme.colors.dark_green};
  }
  &:hover {
    border-color: ${({ theme }) => theme.colors.secondary};
  }
`;

const InactiveButton = styled(BaseButton)`
  border-radius: 0 32px 32px 0;
  background-color: ${({ theme }) => theme.colors.blank_background};
  &.colored {
    background-color: ${({ theme }) => theme.colors.darker_red};
  }
  &:hover {
    border-color: ${({ theme }) => theme.colors.secondary};
  }
`;

const ToggleButtons: React.FC<ToggleProps> = ({ activeText = 'Active', inactiveText = 'Inactive', tooltip, initialValue = false, onChange, disabled }) => {
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
      <Container>
        <ActiveButton className={isActive ? 'colored' : ''} onClick={handleToggle} disabled={disabled}>
          <Image src='/icons/common/circled-check.svg' alt='' width={16} height={16} />
          {activeText}
        </ActiveButton>
        <InactiveButton className={isActive ? '' : 'colored'} onClick={handleToggle} disabled={disabled}>
          <Image src='/icons/common/circled-cross.svg' alt='' width={16} height={16} />
          {inactiveText}
        </InactiveButton>
      </Container>

      {tooltip && <Image src='/icons/common/info.svg' alt='' width={16} height={16} style={{ margin: '0 8px' }} />}
    </Tooltip>
  );
};

export { ToggleButtons };
