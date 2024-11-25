import Image from 'next/image';
import { Text } from '../text';
import theme from '@/styles/theme';
import { Tooltip } from '../tooltip';
import styled from 'styled-components';
import React, { useEffect, useState } from 'react';

interface CheckboxProps {
  title?: string;
  titleColor?: React.CSSProperties['color'];
  tooltip?: string;
  initialValue?: boolean;
  onChange?: (value: boolean) => void;
  disabled?: boolean;
  style?: React.CSSProperties;
}

const Container = styled.div<{ $disabled?: CheckboxProps['disabled'] }>`
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: ${({ $disabled }) => ($disabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ $disabled }) => ($disabled ? 0.6 : 1)};
`;

const CheckboxWrapper = styled.div<{ $isChecked: boolean; $disabled?: CheckboxProps['disabled'] }>`
  width: 18px;
  height: 18px;
  border-radius: 6px;
  border: ${({ $isChecked }) => ($isChecked ? '1px dashed transparent' : '1px dashed rgba(249, 249, 249, 0.4)')};
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: ${({ $isChecked, theme }) => ($isChecked ? theme.colors.majestic_blue : 'transparent')};
  pointer-events: ${({ $disabled }) => ($disabled ? 'none' : 'auto')};
  transition: border 0.3s, background-color 0.3s;
`;

const Checkbox: React.FC<CheckboxProps> = ({ title, titleColor, tooltip, initialValue = false, onChange, disabled, style }) => {
  const [isChecked, setIsChecked] = useState(initialValue);

  useEffect(() => {
    if (isChecked !== initialValue) setIsChecked(initialValue);
  }, [isChecked, initialValue]);

  const handleToggle: React.MouseEventHandler<HTMLDivElement> = (e) => {
    if (disabled) return;

    e.stopPropagation();

    setIsChecked((prev) => {
      const newValue = !prev;
      if (onChange) onChange(newValue);
      return newValue;
    });
  };

  return (
    <Container $disabled={disabled} onClick={handleToggle} style={style}>
      <CheckboxWrapper $isChecked={isChecked} $disabled={disabled}>
        {isChecked && <Image src='/icons/common/check.svg' alt='' width={12} height={12} />}
      </CheckboxWrapper>

      {title && (
        <Tooltip text={tooltip} withIcon>
          <Text size={12} color={titleColor || theme.text.grey} style={{ maxWidth: '90%' }}>
            {title}
          </Text>
        </Tooltip>
      )}
    </Container>
  );
};

export { Checkbox };
