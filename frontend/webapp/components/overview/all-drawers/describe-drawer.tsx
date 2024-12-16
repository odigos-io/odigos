import React from 'react';
import { OdigosLogo } from '@/assets';
import styled from 'styled-components';
import { useDescribeOdigos } from '@/hooks';
import { DATA_CARDS, safeJsonStringify } from '@/utils';
import { DataCard, DataCardFieldTypes } from '@/reuseable-components';
import OverviewDrawer from '@/containers/main/overview/overview-drawer';

interface Props {}

const DataContainer = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

export const DescribeDrawer: React.FC<Props> = () => {
  const { data: describe } = useDescribeOdigos();

  return (
    <OverviewDrawer title={DATA_CARDS.DESCRIBE_ODIGOS} titleTooltip='' icon={OdigosLogo}>
      <DataContainer>
        <DataCard
          title=''
          data={[
            {
              type: DataCardFieldTypes.CODE,
              width: 'inherit',
              value: JSON.stringify({ language: 'json', code: safeJsonStringify(describe) }),
            },
          ]}
        />
      </DataContainer>
    </OverviewDrawer>
  );
};
