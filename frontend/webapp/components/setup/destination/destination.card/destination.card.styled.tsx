import styled from 'styled-components';

export const DestinationCardWrapper = styled.div`
  padding: 1vw 1vh;
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 1vh;
  cursor: pointer;
  border: 1px solid transparent;
  &:hover {
    border-radius: 24px;
    border: ${({ theme }) => `1px solid  ${theme.colors.secondary}`};
  }
  @media screen and (max-width: 1200px) {
    flex-direction: row;
    padding-left: 2vw;
  }
`;

export const DestinationCardContentWrapper = styled.div`
  display: flex;
  align-items: center;
  flex-direction: column;
  @media screen and (max-width: 1200px) {
    align-items: flex-start;
    padding-left: 1vw;
  }
`;

export const ApplicationNameWrapper = styled.div`
  display: flex;
  align-items: center;
  text-overflow: ellipsis;
  height: 50px;
  @media screen and (max-width: 1200px) {
    height: 30px;
  }
`;
