import React, { useState } from 'react';
import styled from 'styled-components';
import { useDescribeOdigos } from '@/hooks';
import { DATA_CARDS, safeJsonStringify } from '@/utils';
import { CodeBracketsIcon, CodeIcon, ListIcon } from '@/assets';
import OverviewDrawer from '@/containers/main/overview/overview-drawer';
import { DataCard, DataCardFieldTypes, Segment } from '@/reuseable-components';

interface Props {}

const DataContainer = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

export const CliDrawer: React.FC<Props> = () => {
  const { data: describe, restructureForPrettyMode } = useDescribeOdigos();
  const [isPrettyMode, setIsPrettyMode] = useState(true);

  return (
    <OverviewDrawer title='Odigos CLI' icon={CodeBracketsIcon}>
      <DataContainer>
        <DataCard
          title={DATA_CARDS.DESCRIBE_ODIGOS}
          action={
            <Segment
              options={[
                { icon: ListIcon, value: true },
                { icon: CodeIcon, value: false },
              ]}
              selected={isPrettyMode}
              setSelected={setIsPrettyMode}
            />
          }
          data={[
            {
              type: DataCardFieldTypes.CODE,
              value: JSON.stringify({
                language: 'json',
                code: safeJsonStringify(isPrettyMode ? restructureForPrettyMode(describe) : describe),
                pretty: isPrettyMode,
              }),
              width: 'inherit',
            },
          ]}
        />
      </DataContainer>
    </OverviewDrawer>
  );
};
