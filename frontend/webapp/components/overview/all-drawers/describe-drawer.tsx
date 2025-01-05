import React, { useState } from 'react';
import styled from 'styled-components';
import { CodeBracketsIcon } from '@/assets';
import { useDescribeOdigos } from '@/hooks';
import { NOTIFICATION_TYPE, type DescribeOdigos } from '@/types';
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

// This function is used to restructure the data, so that it reflects the output given by "odigos describe" command in the CLI.
// This is not really needed, but it's a nice-to-have feature to make the data more readable.
const restructureForPrettyMode = (code?: DescribeOdigos['describeOdigos']) => {
  if (!code) return {};

  const payload = {
    [code.odigosVersion.name]: code.odigosVersion.value,
    'Number Of Sources': code.numberOfSources,
    'Number Of Destinations': code.numberOfDestinations,
    'Cluster Collector': {},
    'Node Collector': {},
  };

  const mapObjects = (obj: any, objectName: string) => {
    if (typeof obj === 'object' && !!obj?.name) {
      let key = obj.name;
      let val = obj.value;
      if (obj.explain) key += `#tooltip=${obj.explain}`;
      if (!payload[objectName]) payload[objectName] = {};
      payload[objectName][key] = val;
    }
  };

  Object.values(code.clusterCollector).forEach((val) => mapObjects(val, 'Cluster Collector'));
  Object.values(code.nodeCollector).forEach((val) => mapObjects(val, 'Node Collector'));

  return payload;
};

export const DescribeDrawer: React.FC<Props> = () => {
  const { data: describe } = useDescribeOdigos();
  const [isCodeMode, setIsCodeMode] = useState(false);

  return (
    <OverviewDrawer title={DATA_CARDS.DESCRIBE_ODIGOS} icon={CodeBracketsIcon}>
      <DataContainer>
        <DataCard
          title=''
          action={<ToggleCodeComponent isCodeMode={isCodeMode} setIsCodeMode={setIsCodeMode} />}
          data={[
            {
              type: DataCardFieldTypes.CODE,
              value: JSON.stringify({
                language: 'json',
                code: safeJsonStringify(isCodeMode ? describe : restructureForPrettyMode(describe)),
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
