import styled from 'styled-components';

// this is to control modal size + scroll
// note: add-destinations does not use this (yet), because it has a custom sidebar
export const ModalBody = styled.div<{ $isNotModal?: boolean }>`
  width: 640px;
  height: ${({ $isNotModal }) => ($isNotModal ? 'fit-content' : 'calc(100vh - 350px)')};
  margin: ${({ $isNotModal }) => ($isNotModal ? '64px 0 0 0' : '64px 7vw 32px 7vw')};
  overflow-y: scroll;
`;
