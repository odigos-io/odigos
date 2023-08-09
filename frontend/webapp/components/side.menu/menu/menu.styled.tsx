import styled from "styled-components";

export const MenuContainer = styled.div`
  width: 234px;
  height: 100%;
  flex-shrink: 0;
  border-right: 1px solid rgba(255, 255, 255, 0.04);
  background: ${({ theme }) => theme.colors.dark_blue};
`;
export const LogoWrapper = styled.div`
  padding: 24px 16px;
`;

export const MenuItemsWrapper = styled.div`
  padding: 16px 9px;
`;

export const ContactUsWrapper = styled(MenuItemsWrapper)`
  position: absolute;
  bottom: 5%;
`;
