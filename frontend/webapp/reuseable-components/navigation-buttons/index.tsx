import React from 'react';
import Image from 'next/image';
import { Button } from '../button';
import { Tooltip } from '../tooltip';
import { Types } from '@odigos/ui-components';
import styled, { useTheme } from 'styled-components';

export interface NavigationButtonProps {
  label: string;
  icon?: Types.SVG;
  iconSrc?: string;
  tooltip?: string;
  onClick: () => void;
  variant?: 'primary' | 'secondary';
  disabled?: boolean;
}

interface Props {
  buttons: NavigationButtonProps[];
}

const ButtonsContainer = styled.div`
  display: flex;
  gap: 8px;
  align-items: center;
`;

const StyledButton = styled(Button)`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-width: 90px;
`;

export const NavigationButtons: React.FC<Props> = ({ buttons }) => {
  const theme = useTheme();

  const shouldRenderBackButton = ({ button, index }: { button: NavigationButtonProps; index: number }) => {
    return buttons.length > 1 && index === 0 && (button.icon || button.iconSrc);
  };

  const renderButton = ({ button, rotate }: { button: NavigationButtonProps; rotate: number }) => {
    return button.icon ? (
      <button.icon size={14} rotate={rotate} fill={theme.text[button.variant || 'secondary']} />
    ) : button.iconSrc ? (
      <Image src={button.iconSrc} alt={button.label} width={8} height={12} />
    ) : null;
  };

  return (
    <ButtonsContainer>
      {buttons.map((btn, index) => (
        <Tooltip key={index} text={btn.tooltip || ''}>
          <StyledButton key={index} variant={btn.variant || 'secondary'} onClick={btn.onClick} disabled={btn.disabled}>
            {shouldRenderBackButton({ button: btn, index }) && renderButton({ button: btn, rotate: 0 })}
            {btn.label}
            {!shouldRenderBackButton({ button: btn, index }) && renderButton({ button: btn, rotate: 180 })}
          </StyledButton>
        </Tooltip>
      ))}
    </ButtonsContainer>
  );
};
