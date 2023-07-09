import styled from "styled-components";

export const TooltipWrapper = styled.div`
  display: inline-block;
  position: relative;
  display: flex;
`;

export const TooltipContentWrapper = styled.div`
  position: absolute;
  max-width: 150px;
  width: min(100px, 250px);
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  padding: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: ${({ theme }) => theme.colors.dark};
  box-shadow: 0px 6px 13px 0px rgba(0, 0, 0, 0.3);
`;
