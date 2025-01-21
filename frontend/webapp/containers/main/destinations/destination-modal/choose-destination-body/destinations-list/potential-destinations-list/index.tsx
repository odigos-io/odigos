import React from 'react';
import { OdigosLogo } from '@/assets';
import styled from 'styled-components';
import type { DestinationTypeItem, SupportedSignals } from '@/types';
import { usePotentialDestinations } from '@/hooks';
import { DataTab, SectionTitle, SkeletonLoader } from '@/reuseable-components';

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
            // @ts-ignore
            key={`select-potential-destination-${item.type}-${idx}`}
            // @ts-ignore
            data-id={`select-potential-destination-${item.type}`}
            // @ts-ignore
            title={item.displayName}
            // @ts-ignore
            iconSrc={item.imageUrl}
            hoverText='Select'
            // @ts-ignore
            monitors={Object.keys(item.supportedSignals).filter((signal: keyof SupportedSignals) => item.supportedSignals[signal].supported)}
            monitorsWithLabels
            onClick={() => setSelectedItems(item)}
          />
        ))
      )}
    </ListsWrapper>
  );
};
