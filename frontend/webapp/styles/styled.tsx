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
export const ModalBody = styled.div`
  width: 640px;
  height: calc(100vh - 300px);
  margin: 0 15vw;
  padding-top: 64px;
  overflow-y: scroll;
`;
