import React from 'react';
import styled from 'styled-components';
import ActualSourceContent from './actual-source-content';

const SourceDrawer: React.FC = () => {
  return (
    <SourceDrawerContainer>
      <ActualSourceContent />
    </SourceDrawerContainer>
  );
};

export { SourceDrawer };

const SourceDrawerContainer = styled.div`
  padding: 16px;
  overflow-y: auto;
`;
