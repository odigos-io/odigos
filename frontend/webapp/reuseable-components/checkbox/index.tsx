import React, { useEffect, useState } from 'react';
import styled from 'styled-components';
import { Tooltip } from '../tooltip';
import Image from 'next/image';
import { Text } from '../text';

interface CheckboxProps {
  title?: string;
  tooltip?: string;
  initialValue?: boolean;
  onChange?: (value: boolean) => void;
  disabled?: boolean;
}

const Container = styled.div<{ disabled?: boolean }>`
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ disabled }) => (disabled ? 0.6 : 1)};
`;

const CheckboxWrapper = styled.div<{ isChecked: boolean; disabled?: boolean }>`
  width: 18px;
  height: 18px;
  border-radius: 6px;
  border: ${({ isChecked }) =>
    isChecked
      ? '1px dashed transparent'
      : '1px dashed rgba(249, 249, 249, 0.4)'};
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: ${({ isChecked, theme }) =>
    isChecked ? theme.colors.majestic_blue : 'transparent'};
  pointer-events: ${({ disabled }) => (disabled ? 'none' : 'auto')};
  transition: border 0.3s, background-color 0.3s;
`;

const Checkbox: React.FC<CheckboxProps> = ({
  title,
  tooltip,
  initialValue = false,
  onChange,
  disabled,
}) => {
  const [isChecked, setIsChecked] = useState(initialValue);

  useEffect(() => {
    setIsChecked(initialValue);
  }, [initialValue]);

  const handleToggle = () => {
    if (!disabled) {
      const newValue = !isChecked;
      setIsChecked(newValue);
      if (onChange) {
        onChange(newValue);
      }
    }
  };

  return (
    <Container disabled={disabled} onClick={handleToggle}>
      <CheckboxWrapper isChecked={isChecked} disabled={disabled}>
        {isChecked && (
          <Image src="/icons/common/check.svg" alt="" width={12} height={12} />
        )}
      </CheckboxWrapper>
      {title && <Text size={14}>{title}</Text>}
      {tooltip && (
        <Tooltip text={tooltip || ''}>
          <Image src="/icons/common/info.svg" alt="" width={16} height={16} />
        </Tooltip>
      )}
    </Container>
  );
};

export { Checkbox };
