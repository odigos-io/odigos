import styled from "styled-components";

interface CardContainerProps {
  selected?: any;
}

export const CardContainer = styled.div<CardContainerProps>`
  display: inline-flex;
  position: relative;
  width: 272px;
  height: 204px;
  flex-direction: column;
  border-radius: 24px;
  border: ${({ selected, theme }) =>
    `1px solid ${selected ? theme.colors.secondary : theme.colors.dark_blue}`};
  background: ${({ theme }) => theme.colors.dark};
`;
