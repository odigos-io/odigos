import styled from 'styled-components';

export const RelativeContainer = styled.div`
  position: relative;
  max-width: 200px;
`;

export const AbsoluteContainer = styled.div`
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  z-index: 1;
  background-color: ${({ theme }) => theme.colors.dropdown_bg};
  border: ${({ theme }) => `1px solid ${theme.colors.border}`};
  border-radius: 24px;
  width: 420px;
`;
