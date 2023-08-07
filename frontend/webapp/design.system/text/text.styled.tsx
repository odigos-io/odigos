import styled from "styled-components";

export const TextWrapper = styled.p`
  color: ${({ theme }) => theme.text.white};
  margin: 0;
  font-family: ${({ theme }) => theme.font_family.primary}, sans-serif;
  font-size: 16px;
  font-weight: 400;
`;
