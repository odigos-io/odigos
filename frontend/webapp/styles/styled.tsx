import styled from 'styled-components';

export const CenterThis = styled.div`
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
`;

export const Overlay = styled.div`
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(1px);
`;

// this is to control modal size + scroll
// note: add-destinations does not use this (yet), because it has a custom sidebar
export const ModalBody = styled.div<{ $isModal?: boolean }>`
  width: 640px;
  height: ${({ $isModal }) => ($isModal ? 'calc(100vh - 350px)' : 'fit-content')};
  margin: 64px 7vw 32px 7vw;
  overflow-y: scroll;
`;

export const FlexRow = styled.div<{ $gap?: number }>`
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: ${({ $gap = 2 }) => $gap}px;
`;

export const FlexColumn = styled.div<{ $gap?: number }>`
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: ${({ $gap = 2 }) => $gap}px;
`;
