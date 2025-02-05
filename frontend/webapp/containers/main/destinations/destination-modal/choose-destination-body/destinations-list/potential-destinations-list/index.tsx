import React from 'react';
import styled from 'styled-components';
import { OdigosLogo } from '@odigos/ui-icons';
import { SIGNAL_TYPE } from '@odigos/ui-utils';
import { usePotentialDestinations } from '@/hooks';
import { type FetchedDestinationTypeItem } from '@/types';
import { DataTab, SectionTitle, SkeletonLoader } from '@odigos/ui-components';

interface Props {
  setSelectedItems: (item: FetchedDestinationTypeItem) => void;
}

const ListsWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

export const PotentialDestinationsList: React.FC<Props> = ({ setSelectedItems }) => {
  const { data, loading } = usePotentialDestinations();

  if (!data.length && !loading) return null;

  return (
    <ListsWrapper>
      <SectionTitle
        size='small'
        icon={OdigosLogo}
        title='Detected by Odigos'
        description='Odigos detects destinations for which automatic connection is available. All data will be filled out automatically.'
      />
      {loading ? (
        <SkeletonLoader size={1} />
      ) : (
        data.map((item, idx) => (
          <DataTab
            key={`select-potential-destination-${item.type}-${idx}`}
            data-id={`select-potential-destination-${item.type}`}
            title={item.displayName}
            iconSrc={item.imageUrl}
            hoverText='Select'
            monitors={Object.keys(item.supportedSignals).filter((signal: SIGNAL_TYPE) => item.supportedSignals[signal].supported) as SIGNAL_TYPE[]}
            monitorsWithLabels
            onClick={() => setSelectedItems(item)}
          />
        ))
      )}
    </ListsWrapper>
  );
};
