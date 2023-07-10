import styled from "styled-components";

export const SelectedCounterWrapper = styled.div`
  display: flex;
  padding: 4px;
  align-items: center;
  gap: 4px;
  border-radius: 14px;
  background: ${({ theme }) => theme.colors.dark_blue};
`;
