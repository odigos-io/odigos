import styled from 'styled-components';

export const SourcesSectionWrapper = styled.div`
  position: relative;
  height: 81%;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;

  @media screen and (max-height: 650px) {
    height: 72%;
  }
  @media screen and (max-height: 550px) {
    height: 65%;
  }
`;

export const ButtonWrapper = styled.div`
  position: absolute;
  display: flex;
  align-items: center;
  gap: 16px;
  right: 32px;
  top: 40px;
`;
