import styled from 'styled-components';

export const HideScroll = styled.div`
  overflow-y: hidden;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;
`;

export const Overlay = styled.div<{ hideOverlay?: boolean }>`
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(1px);
  opacity: ${({ hideOverlay }) => (hideOverlay ? 0 : 1)};
`;
