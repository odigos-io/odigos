import styled from 'styled-components';

export const MenuContainer = styled.div<{ $isExpanded?: boolean; }>`
  transition: width 0.1s;
  height: 100%;
  border-right: 1px solid rgba(255, 255, 255, 0.04);
  background: ${({ theme }) => theme.colors.dark_blue};
  width: ${({$isExpanded}) => $isExpanded ? 234 : 70}px;
`;

export const LogoWrapper = styled.div`
  cursor: pointer;
  padding: 24px 16px;
  height: 48px;
  position: relative;
  opacity: 0;
  animation: slideInFromLeft 2s forwards;
  @keyframes slideInFromLeft {
    to {
      left: 0; /* Slide in to the final position */
      opacity: 1;
    }
  }
`;

export const MenuItemsWrapper = styled.div`
  padding: 16px 9px;
`;

export const ContactUsWrapper = styled(MenuItemsWrapper)`
  position: absolute;
  bottom: 0%;
`;
