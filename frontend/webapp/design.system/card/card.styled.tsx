import styled from "styled-components";

interface CardContainerProps {
  focus?: any;
}

export const CardContainer = styled.div<CardContainerProps>`
  display: inline-flex;
  position: relative;
  width: 272px;
  height: 204px;
  flex-direction: column;
  border-radius: 24px;
  border: ${({ focus, theme }) =>
    `1px solid ${focus ? theme.colors.secondary : theme.colors.dark_blue}`};
  background: ${({ theme }) => theme.colors.dark};
`;
