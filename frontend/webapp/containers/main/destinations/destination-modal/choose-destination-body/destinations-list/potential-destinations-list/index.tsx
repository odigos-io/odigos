import React from 'react';
import styled from 'styled-components';
import { usePotentialDestinations } from '@/hooks';
import { DataTab, SectionTitle } from '@/reuseable-components';
import { OdigosLogo, SkeletonLoader, Types } from '@odigos/ui-components';
import type { DestinationTypeItem, SupportedSignals } from '@/types';

interface Props {
  setSelectedItems: (item: DestinationTypeItem) => void;
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
            monitors={Object.keys(item.supportedSignals).filter((signal: Types.SIGNAL_TYPE) => item.supportedSignals[signal].supported) as Types.SIGNAL_TYPE[]}
            monitorsWithLabels
            onClick={() => setSelectedItems(item)}
          />
        ))
      )}
    </ListsWrapper>
  );
};
