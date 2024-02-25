import styled from 'styled-components';

export const ActionsListWrapper = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
  padding: 0 24px 24px 24px;
  overflow-y: auto;
  align-items: start;
  max-height: 100%;
  padding-bottom: 220px;
  box-sizing: border-box;
`;

export const DescriptionWrapper = styled.div`
  padding: 24px;
  gap: 4px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
`;

export const LinkWrapper = styled.div`
  width: 100px;
`;

export const ActionCardWrapper = styled.div`
  height: 100%;
  max-height: 220px;
`;
