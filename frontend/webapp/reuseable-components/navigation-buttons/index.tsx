import React from 'react';
import Image from 'next/image';
import { SVG } from '@/assets';
import { Button } from '../button';
import styled from 'styled-components';
import theme from '@/styles/theme';

export interface NavigationButtonProps {
  label: string;
  icon?: SVG;
  iconSrc?: string;
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
      {buttons.map((button, index) => (
        <StyledButton key={index} variant={button.variant || 'secondary'} onClick={button.onClick} disabled={button.disabled}>
          {shouldRenderBackButton({ button, index }) && renderButton({ button, rotate: 0 })}
          {button.label}
          {!shouldRenderBackButton({ button, index }) && renderButton({ button, rotate: 180 })}
        </StyledButton>
      ))}
    </ButtonsContainer>
  );
};
