import styled from 'styled-components';

export const Body = styled.div`
  padding: 32px 24px 0;
  border-left: 1px solid rgba(249, 249, 249, 0.08);
  min-height: 600px;
  width: 100%;
  min-width: 770px;
`;

export const SideMenuWrapper = styled.div`
  padding: 32px;
  width: 196px;
  @media (max-width: 1050px) {
    display: none;
  }
`;

export const Container = styled.div`
  display: flex;
`;
