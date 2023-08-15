import styled from "styled-components";

export const DestinationCardWrapper = styled.div`
  padding: 24px;
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 14px;
  cursor: pointer;
  border: 1px solid transparent;
  &:hover {
    border-radius: 24px;
    border: ${({ theme }) => `1px solid  ${theme.colors.secondary}`};
  }
`;

export const ApplicationNameWrapper = styled.div`
  display: flex;
  align-items: center;
  text-overflow: ellipsis;
  max-width: 224px;
  height: 40px;
`;
