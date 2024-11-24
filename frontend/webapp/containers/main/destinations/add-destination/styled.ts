import styled from 'styled-components';

export const SideMenuWrapper = styled.div`
  border-right: 1px solid rgba(249, 249, 249, 0.08);
  padding: 32px;
  width: 196px;
  @media (max-width: 1050px) {
    display: none;
  }
`;

export const Container = styled.div`
  display: flex;
`;
