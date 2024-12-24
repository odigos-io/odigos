import React, { useEffect, useState } from 'react';
import { Text } from '../text';
import theme from '@/styles/theme';
import { CheckIcon } from '@/assets';
import { Tooltip } from '../tooltip';
import styled from 'styled-components';
import { FlexColumn } from '@/styles';
import { FieldError } from '../field-error';

interface CheckboxProps {
  title?: string;
  titleColor?: React.CSSProperties['color'];
  tooltip?: string;
  value?: boolean;
  onChange?: (value: boolean) => void;
  disabled?: boolean;
  style?: React.CSSProperties;
  errorMessage?: string;
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

export const Checkbox: React.FC<CheckboxProps> = ({ title, titleColor, tooltip, value = false, onChange, disabled, style, errorMessage }) => {
  const [isChecked, setIsChecked] = useState(value);
  useEffect(() => setIsChecked(value), [value]);

  const handleToggle: React.MouseEventHandler<HTMLDivElement> = (e) => {
    if (disabled) return;

    e.stopPropagation();

    if (onChange) onChange(!isChecked);
    else setIsChecked((prev) => !prev);
  };

  return (
    <FlexColumn>
      <Container data-id={`checkbox${!!title ? `-${title}` : ''}`} $disabled={disabled} onClick={handleToggle} style={style}>
        <CheckboxWrapper $isChecked={isChecked} $disabled={disabled}>
          {isChecked && <CheckIcon />}
        </CheckboxWrapper>

        {title && (
          <Tooltip text={tooltip} withIcon>
            <Text size={12} color={titleColor || theme.text.grey} style={{ maxWidth: '90%' }}>
              {title}
            </Text>
          </Tooltip>
        )}
      </Container>

      {!!errorMessage && <FieldError>{errorMessage}</FieldError>}
    </FlexColumn>
  );
};
