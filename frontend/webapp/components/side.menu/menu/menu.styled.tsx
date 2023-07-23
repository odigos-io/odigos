import styled from "styled-components";

export const MenuContainer = styled.div`
  width: 234px;
  height: 100%;
  flex-shrink: 0;
  border-right: 1px solid rgba(255, 255, 255, 0.04);
  background: ${({ theme }) => theme.colors.dark_blue};
`;
