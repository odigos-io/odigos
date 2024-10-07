import React from 'react';
import styled from 'styled-components';
import { CardDetails, ConfiguredDestinationFields } from '@/components';
import { Text } from '@/reuseable-components';

const SourceDrawer: React.FC = () => {
  return (
    <>
      <CardDetails
        data={[
          {
            title: 'key1',
            value: 'value1',
          },
          {
            title: 'key2',
            value: 'value2',
          },
          {
            title: 'key3',
            value: 'value3',
          },
        ]}
      />
    </>
  );
};

export { SourceDrawer };

const SourceDrawerContainer = styled.div`
  display: flex;
  flex-direction: column;
  padding: 16px 24px 24px 24px;
  flex-direction: column;
  align-items: flex-start;
  gap: 16px;
  align-self: stretch;
  border-radius: 24px;
  border: 1px solid ${({ theme }) => theme.colors.border};
`;

const TitleWrapper = styled.div``;
