import styled from 'styled-components';

export const Container = styled.div`
  display: flex;
  justify-content: center;
  max-height: 100%;
  overflow-y: auto;
`;

export const InstrumentationRulesContainer = styled.div`
  margin-top: 24px;
  width: 100%;
  max-width: 1216px;
`;

export const Header = styled.div`
  display: flex;
  justify-content: space-between;
  padding: 0 24px;
  align-items: center;
`;

export const HeaderRight = styled.div`
  display: flex;
  gap: 8px;
  align-items: center;
`;

export const Content = styled.div`
  padding: 20px;
  min-height: 200px;
`;
