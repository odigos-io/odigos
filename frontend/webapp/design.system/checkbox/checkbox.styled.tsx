import { styled } from "styled-components";

export const CheckboxWrapper = styled.div`
  display: flex;
  gap: 8px;
  align-items: center;
  cursor: pointer;
`;

export const Checkbox = styled.span`
  width: 16px;
  height: 16px;
  border: ${({ theme }) => `solid 1px ${theme.colors.light_grey}`};
  border-radius: 4px;
`;
