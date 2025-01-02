import React, { useState } from 'react';
import styled from 'styled-components';
import { CodeBracketsIcon } from '@/assets';
import { useDescribeOdigos } from '@/hooks';
import { DATA_CARDS, safeJsonStringify } from '@/utils';
import { ToggleCodeComponent } from '@/components/common';
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
  const [isCodeMode, setIsCodeMode] = useState(false);

  return (
    <OverviewDrawer title='' icon={CodeBracketsIcon}>
      <DataContainer>
        <DataCard
          title={DATA_CARDS.DESCRIBE_ODIGOS}
          action={<ToggleCodeComponent isCodeMode={isCodeMode} setIsCodeMode={setIsCodeMode} />}
          data={[
            {
              type: DataCardFieldTypes.CODE,
              value: JSON.stringify({
                language: 'json',
                code: safeJsonStringify(describe),
                pretty: !isCodeMode,
              }),
              width: 'inherit',
            },
          ]}
        />
      </DataContainer>
    </OverviewDrawer>
  );
};
