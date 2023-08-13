import styled from 'styled-components';

export const RadioButtonWrapper = styled.div`
  position: absolute;
  right: 16px;
  top: 16px;
`;

export const SourceCardWrapper = styled.div`
  padding: 1vw;
  display: flex;
  align-items: center;
  flex-direction: column;

  cursor: pointer;
  .p {
    cursor: pointer !important;
  }
  @media screen and (max-width: 1200px) {
    flex-direction: row;
  }
`;

export const SourceTextWrapper = styled.div`
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 1vh;
  @media screen and (max-width: 1200px) {
    align-items: flex-start;
    margin-left: 1vw;
  }
`;

export const ApplicationNameWrapper = styled.div`
  display: inline-block;
  text-overflow: ellipsis;
  max-width: 224px;
  @media screen and (max-width: 1250px) {
    max-width: 154px;
  }
  @media screen and (max-width: 1150px) {
    max-width: 224px;
  }
`;
