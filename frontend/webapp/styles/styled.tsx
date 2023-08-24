import styled from 'styled-components';

export const HideScroll = styled.div`
  overflow-y: hidden;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;
`;
