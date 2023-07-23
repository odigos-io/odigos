import styled from "styled-components";

export const NotificationContainer = styled.div`
  position: fixed;
  top: 3%;
  right: 3%;
`;

export const StyledNotification = styled.div`
  display: flex;
  height: 24px;
  padding: 6px 16px 6px 8px;
  /* width: 100%; */
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border-radius: 8px;
  border: ${({ theme }) => `1px solid ${theme.colors.secondary}`};
  background: ${({ theme }) => theme.colors.dark_blue};
  svg {
    cursor: pointer;
  }
`;
