import styled from "styled-components";

interface CardContainerProps {
  focus?: boolean;
}

export const CardContainer = styled.div<CardContainerProps>`
  display: inline-flex;
  position: relative;
  width: 272px;
  height: 204px;
  flex-direction: column;
  border-radius: 24px;
  border: ${({ focus }) => `1px solid ${focus ? "#96F2FF" : "#203548"}`};
  background: var(--dark-mode-dark-1, #0a1824);
`;
