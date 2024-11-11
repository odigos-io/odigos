import styled from 'styled-components';

export const RelativeContainer = styled.div`
  position: relative;
  width: 200px;
`;

export const CardWrapper = styled.div`
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  z-index: 10;
  background-color: ${({ theme }) => theme.colors.dropdown_bg};
  border: ${({ theme }) => `1px solid ${theme.colors.border}`};
  border-radius: 24px;
  width: 440px;
`;

export const CardContent = styled.div`
  display: flex;
  flex-direction: column;
`;
